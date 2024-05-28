// Package viamrtsp implements RTSP camera support in a Viam module
package viamrtsp

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/bluenviron/gortsplib/v4/pkg/format/rtph264"
	"github.com/bluenviron/gortsplib/v4/pkg/format/rtph265"
	"github.com/bluenviron/gortsplib/v4/pkg/liberrors"
	"github.com/bluenviron/mediacommon/pkg/codecs/h264"
	"github.com/erh/viamrtsp/formatprocessor"
	"github.com/pion/rtp"
	"github.com/pkg/errors"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/components/camera/rtppassthrough"
	"go.viam.com/rdk/gostream"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/rimage/transform"
	rutils "go.viam.com/rdk/utils"
	"go.viam.com/utils"
)

var (
	family = resource.ModelNamespace("seanorg").WithFamily("seanviamrtsp")
	// ModelAgnostic selects the best available codec.
	ModelAgnostic = family.WithModel("rtsp")
	// ModelH264 uses the h264 codec.
	ModelH264 = family.WithModel("rtsp-h264")
	// ModelH265 uses the h265 codec.
	ModelH265 = family.WithModel("rtsp-h265")
	// ModelMJPEG uses the mjpeg codec.
	ModelMJPEG = family.WithModel("rtsp-mjpeg")
	// Models is a slice containing the above available models.
	Models = []resource.Model{ModelAgnostic, ModelH264, ModelH265, ModelMJPEG}
	// ErrH264PassthroughNotEnabled is an error indicating H264 passthrough is not enabled.
	ErrH264PassthroughNotEnabled = errors.New("H264 passthrough is not enabled")
)

func init() {
	for _, model := range Models {
		resource.RegisterComponent(camera.API, model, resource.Registration[camera.Camera, *Config]{
			Constructor: newRTSPCamera,
		})
	}
}

// Config are the config attributes for an RTSP camera model.
type Config struct {
	Address          string                             `json:"rtsp_address"`
	RTPPassthrough   bool                               `json:"rtp_passthrough"`
	IntrinsicParams  *transform.PinholeCameraIntrinsics `json:"intrinsic_parameters,omitempty"`
	DistortionParams *transform.BrownConrady            `json:"distortion_parameters,omitempty"`
}

// CodecFormat contains a pointer to a format and the corresponding FFmpeg codec.
type codecFormat struct {
	formatPointer interface{}
	codec         videoCodec
}

// Validate checks to see if the attributes of the model are valid.
func (conf *Config) Validate(path string) ([]string, error) {
	_, err := base.ParseURL(conf.Address)
	if err != nil {
		return nil, fmt.Errorf("invalid address '%s' for component at path '%s': %w", conf.Address, path, err)
	}
	if conf.IntrinsicParams != nil {
		if err := conf.IntrinsicParams.CheckValid(); err != nil {
			return nil, fmt.Errorf("invalid intrinsic parameters for component at path '%s': %w", path, err)
		}
	}
	if conf.DistortionParams != nil {
		if err := conf.DistortionParams.CheckValid(); err != nil {
			return nil, fmt.Errorf("invalid distortion parameters for component at path '%s': %w", path, err)
		}
	}

	return nil, nil
}

type (
	unitSubscriberFunc func(formatprocessor.Unit)
	bufAndCB           struct {
		cb  unitSubscriberFunc
		buf *rtppassthrough.Buffer
	}
)

// rtspCamera contains the rtsp client, and the reader function that fulfills the camera interface.
type rtspCamera struct {
	model resource.Model
	gostream.VideoReader
	u *base.URL

	client     *gortsplib.Client
	rawDecoder *decoder

	cancelCtx  context.Context
	cancelFunc context.CancelFunc

	activeBackgroundWorkers sync.WaitGroup

	latestFrame atomic.Pointer[image.Image]

	logger logging.Logger

	rtpPassthrough              bool
	currentCodec                atomic.Int64
	rtpPassthroughCtx           context.Context
	rtpPassthroughCancelCauseFn context.CancelCauseFunc

	subsMu       sync.RWMutex
	bufAndCBByID map[rtppassthrough.SubscriptionID]bufAndCB
}

// Close closes the camera. It always returns nil, but because of Close() interface, it needs to return an error.
func (rc *rtspCamera) Close(_ context.Context) error {
	rc.cancelFunc()
	rc.unsubscribeAll()
	rc.activeBackgroundWorkers.Wait()
	rc.closeConnection()
	return nil
}

// clientReconnectBackgroundWorker checks every 5 sec to see if the client is connected to the server, and reconnects if not.
func (rc *rtspCamera) clientReconnectBackgroundWorker(codecInfo videoCodec) {
	rc.activeBackgroundWorkers.Add(1)
	utils.ManagedGo(func() {
		for utils.SelectContextOrWait(rc.cancelCtx, 5*time.Second) {
			badState := false

			// use an OPTIONS request to see if the server is still responding to requests
			if rc.client == nil {
				badState = true
			} else {
				res, err := rc.client.Options(rc.u)
				// Nick S:
				// This error happens all the time on hardware we need to support & does not affect
				// the performance of camera streaming. As a result, we ignore this error specifically
				var errClientInvalidState liberrors.ErrClientInvalidState
				if err != nil && !errors.As(err, &errClientInvalidState) {
					rc.logger.Warnf("The rtsp client encountered an error, trying to reconnect to %s, err: %s", rc.u, err)
					badState = true
				} else if res != nil && res.StatusCode != base.StatusOK {
					rc.logger.Warnf("The rtsp server responded with non-OK status url: %s, status_code: %d", rc.u, res.StatusCode)
					badState = true
				}
			}

			if badState {
				if err := rc.reconnectClient(codecInfo); err != nil {
					rc.logger.Warnf("cannot reconnect to rtsp server err: %s", err.Error())
				} else {
					rc.logger.Infof("reconnected to rtsp server url: %s", rc.u)
				}
			}
		}
	}, rc.activeBackgroundWorkers.Done)
}

func (rc *rtspCamera) closeConnection() {
	if rc.client != nil {
		rc.client.Close()
		rc.client = nil
	}
	rc.currentCodec.Store(0)
	if rc.rawDecoder != nil {
		rc.rawDecoder.close()
		rc.rawDecoder = nil
	}
}

// reconnectClient reconnects the RTSP client to the streaming server by closing the old one and starting a new one.
func (rc *rtspCamera) reconnectClient(codecInfo videoCodec) error {
	rc.logger.Warnf("reconnectClient called with codec: %s", codecInfo)

	rc.closeConnection()

	// replace the client with a new one, but close it if setup is not successful
	rc.client = &gortsplib.Client{}
	rc.client.OnPacketLost = func(err error) {
		rc.logger.Debugf("OnPacketLost: err: %s", err)
	}
	rc.client.OnTransportSwitch = func(err error) {
		rc.logger.Debugf("OnTransportSwitch: err: %s", err)
	}
	rc.client.OnDecodeError = func(err error) {
		rc.logger.Debugf("OnDecodeError: err: %s", err)
	}

	if err := rc.client.Start(rc.u.Scheme, rc.u.Host); err != nil {
		return errors.Wrapf(err, "when calling RTSP START on Scheme: %s, Host: %s", rc.u.Scheme, rc.u.Host)
	}

	var clientSuccessful bool
	defer func() {
		if !clientSuccessful {
			rc.closeConnection()
		}
	}()

	session, _, err := rc.client.Describe(rc.u)
	if err != nil {
		return errors.Wrapf(err, "when calling RTSP DESCRIBE on %s", rc.u)
	}

	if codecInfo == Agnostic {
		codecInfo = getAvailableCodec(session)
	}

	switch codecInfo {
	case H264:
		rc.logger.Info("setting up H264 decoder")
		if err := rc.initH264(session); err != nil {
			return err
		}
	case H265:
		rc.logger.Info("setting up H265 decoder")
		if err := rc.initH265(session); err != nil {
			return err
		}
	case MJPEG:
		rc.logger.Info("setting up MJPEG decoder")
		if err := rc.initMJPEG(session); err != nil {
			return err
		}
	case Unknown:
		return errors.New("codecInfo should not be Unknown after getting stream info")
	case Agnostic:
		return errors.New("codecInfo should not be Agnostic after getting stream info")
	default:
		return errors.Errorf("codec not supported %v", codecInfo)
	}

	if _, err := rc.client.Play(nil); err != nil {
		return err
	}
	clientSuccessful = true
	rc.currentCodec.Store(int64(codecInfo))
	// if after reconnecting we no longer support rtp_passthrough
	// terminate all subscription
	// otherwise, let any remaining subscriptions continue
	// NOTE: We should test if subscriptions ALWAY recover after
	// reconnecting. If not, we might want to terminate all subscriptions
	// regardless of whether or not passthrough is supported so that
	// subscribers can request new subscriptions.
	if err := rc.validateSupportsPassthrough(); err != nil {
		rc.unsubscribeAll()
	}
	return nil
}

// initH264 initializes the H264 decoder and sets up the client to receive H264 packets.
func (rc *rtspCamera) initH264(session *description.Session) (err error) {
	// setup RTP/H264 -> H264 decoder
	var f *format.H264

	media := session.FindFormat(&f)
	if media == nil {
		rc.logger.Warn("tracks available")
		for _, x := range session.Medias {
			rc.logger.Warnf("\t %v", x)
		}
		return errors.New("h264 track not found")
	}

	// setup RTP/H264 -> H264 decoder
	rtpDec, err := f.CreateDecoder()
	if err != nil {
		return errors.Wrap(err, "creating H264 RTP decoder")
	}

	// setup H264 -> raw frames decoder
	rc.rawDecoder, err = newH264Decoder(rc.logger)
	if err != nil {
		return errors.Wrap(err, "creating H264 raw decoder")
	}

	// if SPS and PPS are present into the SDP, send them to the decoder
	initialSPSAndPPS := [][]byte{}
	if f.SPS != nil {
		initialSPSAndPPS = append(initialSPSAndPPS, f.SPS)
	} else {
		rc.logger.Warn("no initial SPS found in H264 format")
	}
	if f.PPS != nil {
		initialSPSAndPPS = append(initialSPSAndPPS, f.PPS)
	} else {
		rc.logger.Warn("no initial PPS found in H264 format")
	}

	var receivedFirstIDR bool
	storeImage := func(pkt *rtp.Packet) {
		au, err := rtpDec.Decode(pkt)
		if err != nil {
			if !errors.Is(err, rtph264.ErrNonStartingPacketAndNoPrevious) && !errors.Is(err, rtph264.ErrMorePacketsNeeded) {
				rc.logger.Debugf("error decoding(1) h264 rstp stream %w", err)
			}
			return
		}

		if !receivedFirstIDR && h264.IDRPresent(au) {
			rc.logger.Debug("adding initial SPS & PPS")
			receivedFirstIDR = true
			au = append(initialSPSAndPPS, au...)
		}

		rc.storeH264Frame(au)
	}

	onPacketRTP := func(pkt *rtp.Packet) {
		storeImage(pkt)
	}

	if rc.rtpPassthrough {
		fp, err := formatprocessor.New(1472, f, true)
		if err != nil {
			return errors.Wrap(err, "unable to create new h264 rtp formatprocessor")
		}

		publishToWebRTC := func(pkt *rtp.Packet) {
			pts, ok := rc.client.PacketPTS(media, pkt)
			if !ok {
				return
			}
			ntp := time.Now()
			u, err := fp.ProcessRTPPacket(pkt, ntp, pts, true)
			if err != nil {
				rc.logger.Debug(err.Error())
				return
			}
			rc.subsMu.RLock()
			defer rc.subsMu.RUnlock()
			if len(rc.bufAndCBByID) == 0 {
				return
			}

			// Publish the newly received packet Unit to all subscribers
			for _, bufAndCB := range rc.bufAndCBByID {
				if err := bufAndCB.buf.Publish(func() { bufAndCB.cb(u) }); err != nil {
					rc.logger.Debug("RTP packet dropped due to %s", err.Error())
				}
			}
		}

		onPacketRTP = func(pkt *rtp.Packet) {
			publishToWebRTC(pkt)
			storeImage(pkt)
		}
	}

	_, err = rc.client.Setup(session.BaseURL, media, 0, 0)
	if err != nil {
		return errors.Wrapf(err, "when calling RTSP Setup on %s for H264", session.BaseURL)
	}

	rc.client.OnPacketRTP(media, f, onPacketRTP)

	return nil
}

// initH265 initializes the H265 decoder and sets up the client to receive H265 packets.
func (rc *rtspCamera) initH265(session *description.Session) (err error) {
	if rc.rtpPassthrough {
		rc.logger.Warn("rtp_passthrough is only supported for H264 codec. rtp_passthrough features disabled due to H265 RTSP track")
	}
	var f *format.H265

	media := session.FindFormat(&f)
	if media == nil {
		rc.logger.Warn("tracks available")
		for _, x := range session.Medias {
			rc.logger.Warnf("\t %v", x)
		}
		return errors.New("h265 track not found")
	}

	rtpDec, err := f.CreateDecoder()
	if err != nil {
		return errors.Wrap(err, "creating H265 RTP decoder")
	}

	rc.rawDecoder, err = newH265Decoder(rc.logger)
	if err != nil {
		return errors.Wrap(err, "creating H265 raw decoder")
	}

	// For H.265, handle VPS, SPS, and PPS
	if f.VPS != nil {
		//nolint:gosec
		rc.rawDecoder.decode(f.VPS)
	} else {
		rc.logger.Warn("no VPS found in H265 format")
	}

	if f.SPS != nil {
		//nolint:gosec
		rc.rawDecoder.decode(f.SPS)
	} else {
		rc.logger.Warn("no SPS found in H265 format")
	}

	if f.PPS != nil {
		//nolint:gosec
		rc.rawDecoder.decode(f.PPS)
	} else {
		rc.logger.Warn("no PPS found in H265 format")
	}

	_, err = rc.client.Setup(session.BaseURL, media, 0, 0)
	if err != nil {
		return errors.Wrapf(err, "when calling RTSP Setup on %s for H265", session.BaseURL)
	}

	// On packet retreival, turn it into an image, and store it in shared memory
	rc.client.OnPacketRTP(media, f, func(pkt *rtp.Packet) {
		// Extract access units from RTP packets
		au, err := rtpDec.Decode(pkt)
		if err != nil {
			if !errors.Is(err, rtph265.ErrNonStartingPacketAndNoPrevious) && !errors.Is(err, rtph265.ErrMorePacketsNeeded) {
				rc.logger.Debugf("error decoding(1) h265 rstp stream %w", err)
			}
			return
		}

		for _, nalu := range au {
			lastImage, err := rc.rawDecoder.decode(nalu)
			if err != nil {
				rc.logger.Debugf("error decoding(2) h265 rtsp stream err: %s", err.Error())
				return
			}

			if lastImage != nil {
				rc.latestFrame.Store(&lastImage)
			}
		}
	})

	return nil
}

// initMJPEG initializes the MJPEG decoder and sets up the client to receive JPEG frames.
func (rc *rtspCamera) initMJPEG(session *description.Session) error {
	if rc.rtpPassthrough {
		rc.logger.Warn("rtp_passthrough is only supported for H264 codec. rtp_passthrough features disabled due to MJPEG RTSP track")
	}
	var f *format.MJPEG
	media := session.FindFormat(&f)
	if media == nil {
		rc.logger.Warn("tracks available")
		for _, x := range session.Medias {
			rc.logger.Warnf("\t %v", x)
		}
		return errors.New("MJPEG track not found")
	}

	mjpegDecoder, err := f.CreateDecoder()
	if err != nil {
		return errors.Wrap(err, "creating MJPEG RTP decoder")
	}

	_, err = rc.client.Setup(session.BaseURL, media, 0, 0)
	if err != nil {
		return errors.Wrapf(err, "when calling RTSP Setup on %s for MJPEG", session.BaseURL)
	}

	rc.client.OnPacketRTP(media, f, func(pkt *rtp.Packet) {
		frame, err := mjpegDecoder.Decode(pkt)
		if err != nil {
			return
		}
		if frame == nil {
			return
		}

		img, err := jpeg.Decode(bytes.NewReader(frame))
		if err != nil {
			rc.logger.Debugf("error converting MJPEG frame to image err: %s", err.Error())
			return
		}

		rc.latestFrame.Store(&img)
	})

	return nil
}

// SubscribeRTP registers the PacketCallback which will be called when there are new packets.
// NOTE: Packets may be dropped before calling packetsCB if the rate new packets are received by
// the rtppassthrough.Source is greater than the rate the subscriber consumes them.
func (rc *rtspCamera) SubscribeRTP(
	_ context.Context,
	bufferSize int,
	packetsCB rtppassthrough.PacketCallback,
) (rtppassthrough.Subscription, error) {
	if err := rc.validateSupportsPassthrough(); err != nil {
		rc.logger.Debug(err.Error())
		return rtppassthrough.NilSubscription, ErrH264PassthroughNotEnabled
	}

	sub, buf, err := rtppassthrough.NewSubscription(bufferSize)
	if err != nil {
		return rtppassthrough.NilSubscription, err
	}
	g := rutils.NewGuard(func() {
		buf.Close()
	})
	defer g.OnFail()

	webrtcPayloadMaxSize := 1188 // 1200 - 12 (RTP header)
	encoder := &rtph264.Encoder{
		PayloadType:    96,
		PayloadMaxSize: webrtcPayloadMaxSize,
	}

	if err := encoder.Init(); err != nil {
		return rtppassthrough.NilSubscription, err
	}

	var firstReceived bool
	var lastPTS time.Duration
	// OnPacketRTP will call this unitSubscriberFunc for all subscribers.
	// unitSubscriberFunc will then convert the Unit into a slice of
	// WebRTC compliant RTP packets & call packetsCB, which will
	// allow the caller of SubscribeRTP to handle the packets.
	// This is intended to free the SubscribeRTP caller from needing
	// to care about how to transform RTSP compliant RTP packets into
	// WebRTC compliant RTP packets.
	// Inspired by https://github.com/bluenviron/mediamtx/blob/main/internal/servers/webrtc/session.go#L185
	unitSubscriberFunc := func(u formatprocessor.Unit) {
		if err := rc.rtpPassthroughCtx.Err(); err != nil {
			return
		}

		tunit, ok := u.(*formatprocessor.H264)
		if !ok {
			err := errors.New("(*unit.H264) type conversion error")
			rc.logger.Error(err.Error())
			rc.rtpPassthroughCancelCauseFn(err)

			// unsubscribeAll() needs to be run in another goroutine as it will call Close() on sub which
			// will try to take a lock which has already been taken while unitSubscriberFunc is executing
			rc.activeBackgroundWorkers.Add(1)
			utils.ManagedGo(rc.unsubscribeAll, rc.activeBackgroundWorkers.Done)
			return
		}

		// If we have no AUs we can't encode packets.
		if tunit.AU == nil {
			return
		}

		if !firstReceived {
			firstReceived = true
		} else if tunit.PTS < lastPTS {
			err := errors.New("WebRTC doesn't support H264 streams with B-frames")
			rc.logger.Error(err.Error())
			rc.rtpPassthroughCancelCauseFn(err)

			// unsubscribeAll() needs to be run in another goroutine as unsubscribeAll() will call Close() on sub which
			// will try to take a lock which has already been taken while unitSubscriberFunc is executing
			rc.activeBackgroundWorkers.Add(1)
			utils.ManagedGo(rc.unsubscribeAll, rc.activeBackgroundWorkers.Done)
			return
		}
		lastPTS = tunit.PTS

		pkts, err := encoder.Encode(tunit.AU)
		if err != nil {
			// If there is an Encode error we just drop the packets.
			return
		}

		if len(pkts) == 0 {
			// If no packets can be encoded from the AU, there is no need to call the subscriber's callback.
			return
		}

		for _, pkt := range pkts {
			pkt.Timestamp += tunit.RTPPackets[0].Timestamp
		}

		packetsCB(pkts)
	}

	rc.subsMu.Lock()
	defer rc.subsMu.Unlock()

	rc.bufAndCBByID[sub.ID] = bufAndCB{
		cb:  unitSubscriberFunc,
		buf: buf,
	}
	buf.Start()
	g.Success()
	return sub, nil
}

// Unsubscribe deregisters the Subscription's callback.
func (rc *rtspCamera) Unsubscribe(_ context.Context, id rtppassthrough.SubscriptionID) error {
	rc.subsMu.Lock()
	defer rc.subsMu.Unlock()
	bufAndCB, ok := rc.bufAndCBByID[id]
	if !ok {
		return errors.New("id not found")
	}
	delete(rc.bufAndCBByID, id)
	bufAndCB.buf.Close()
	return nil
}

func newRTSPCamera(ctx context.Context, _ resource.Dependencies, conf resource.Config, logger logging.Logger) (camera.Camera, error) {
	newConf, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	u, err := base.ParseURL(newConf.Address)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	rtpPassthroughCtx, rtpPassthroughCancelCauseFn := context.WithCancelCause(context.Background())
	rc := &rtspCamera{
		model:                       conf.Model,
		u:                           u,
		rtpPassthrough:              newConf.RTPPassthrough,
		bufAndCBByID:                make(map[rtppassthrough.SubscriptionID]bufAndCB),
		rtpPassthroughCtx:           rtpPassthroughCtx,
		rtpPassthroughCancelCauseFn: rtpPassthroughCancelCauseFn,
		logger:                      logger,
	}
	codecInfo, err := modelToCodec(conf.Model)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	err = rc.reconnectClient(codecInfo)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	cancelCtx, cancel := context.WithCancel(context.Background())
	reader := gostream.VideoReaderFunc(func(_ context.Context) (image.Image, func(), error) {
		latest := rc.latestFrame.Load()
		if latest == nil {
			return nil, func() {}, errors.New("no frame yet")
		}
		return *latest, func() {}, nil
	})
	rc.VideoReader = reader
	rc.cancelCtx = cancelCtx
	rc.cancelFunc = cancel
	cameraModel := camera.NewPinholeModelWithBrownConradyDistortion(newConf.IntrinsicParams, newConf.DistortionParams)
	rc.clientReconnectBackgroundWorker(codecInfo)
	src, err := camera.NewVideoSourceFromReader(ctx, rc, &cameraModel, camera.ColorStream)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	return camera.FromVideoSource(conf.ResourceName(), src, logger), nil
}

func (rc *rtspCamera) unsubscribeAll() {
	rc.subsMu.Lock()
	defer rc.subsMu.Unlock()
	for id, bufAndCB := range rc.bufAndCBByID {
		delete(rc.bufAndCBByID, id)
		bufAndCB.buf.Close()
	}
}

func (rc *rtspCamera) validateSupportsPassthrough() error {
	if !rc.rtpPassthrough {
		return errors.New("rtp_passthrough not enabled in config")
	}
	modelSupportsPassthrough := rc.model == ModelAgnostic || rc.model == ModelH264
	if !modelSupportsPassthrough {
		return fmt.Errorf("model %s does not support rtp_passthrough", rc.model.Name)
	}

	currentCodec := videoCodec(rc.currentCodec.Load())
	if currentCodec != H264 {
		return fmt.Errorf("rtp_passthrough only supported for H264 codec, current codec is: %s", currentCodec)
	}

	if err := context.Cause(rc.rtpPassthroughCtx); err != nil {
		return errors.Wrap(err, "rtp_passthrough was determined to not be supported at runtime due to")
	}

	return nil
}

func modelToCodec(model resource.Model) (videoCodec, error) {
	switch model {
	case ModelAgnostic:
		return Agnostic, nil
	case ModelH264:
		return H264, nil
	case ModelH265:
		return H265, nil
	case ModelMJPEG:
		return MJPEG, nil
	default:
		return Unknown, fmt.Errorf("model '%s' has unspecified codec handling", model.Name)
	}
}

// getAvailableCodec determines the first supported codec from a session's SDP data
// returning Unknown if none are found.
func getAvailableCodec(session *description.Session) videoCodec {
	var h264 *format.H264
	var h265 *format.H265
	var mjpeg *format.MJPEG

	// List of formats/codecs in priority order
	codecFormats := []codecFormat{
		{&h264, H264},
		{&h265, H265},
		{&mjpeg, MJPEG},
	}

	for _, codecFormat := range codecFormats {
		if session.FindFormat(codecFormat.formatPointer) != nil {
			return codecFormat.codec
		}
	}

	return Unknown
}

func (rc *rtspCamera) storeH264Frame(au [][]byte) {
	naluIndex := 0
	for naluIndex < len(au) {
		nalu := au[naluIndex]
		if isCompactableH264(nalu) {
			// if the NALU is a compactable type, compact it, feed it into the decoder & skip
			// the NALUs that were compacted.
			// We do this so that the libav functions the decoder uses under the hood don't log
			// spam error messages (which happens when it is fed SPS or PPS without an IDR
			nalu, nalusCompacted := rc.compactH264SPSAndPPSAndIDR(au[naluIndex:])
			if err := rc.decodeAndStore(nalu); err != nil {
				rc.logger.Debugf("error decoding(2) h264 rtsp stream  %s", err.Error())
				return
			}
			naluIndex += nalusCompacted
			continue
		}

		// otherwise feed in each non compactable NALU into the decoder
		if err := rc.decodeAndStore(nalu); err != nil {
			rc.logger.Debugf("error decoding(2) h264 rtsp stream  %s", err.Error())
			return
		}
		naluIndex++
	}
}

func (rc *rtspCamera) compactH264SPSAndPPSAndIDR(au [][]byte) ([]byte, int) {
	compactedNALU, numCompacted := []byte{}, 0
	for _, nalu := range au {
		if !isCompactableH264(nalu) {
			// return once we hit a non SPS, PPS or IDR message
			return compactedNALU, numCompacted
		}
		// If this is the first iteration, don't add the start code
		// as the first nalu has not been written yet
		if len(compactedNALU) > 0 {
			startCode := H2645StartCode()
			compactedNALU = append(compactedNALU, startCode...)
		}
		compactedNALU = append(compactedNALU, nalu...)
		numCompacted++
	}
	return compactedNALU, numCompacted
}

// H2645StartCode is start code byte sequence for H264/H265 NALs.
func H2645StartCode() []byte {
	return []uint8{0x00, 0x00, 0x00, 0x01}
}

func (rc *rtspCamera) decodeAndStore(nalu []byte) error {
	image, err := rc.rawDecoder.decode(nalu)
	if err != nil {
		return err
	}
	if image != nil {
		rc.latestFrame.Store(&image)
	}
	return nil
}

func naluType(nalu []byte) h264.NALUType {
	return h264.NALUType(nalu[0] & 0x1F)
}

func isCompactableH264(nalu []byte) bool {
	typ := naluType(nalu)
	return typ == h264.NALUTypeSPS || typ == h264.NALUTypePPS || typ == h264.NALUTypeIDR
}
