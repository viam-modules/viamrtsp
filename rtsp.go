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

	"github.com/aler9/gortsplib/v2/pkg/media"
	"github.com/bluenviron/gortsplib/v3/pkg/formats"
	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/bluenviron/gortsplib/v4/pkg/format/rtph264"
	"github.com/bluenviron/gortsplib/v4/pkg/format/rtph265"
	"github.com/bluenviron/gortsplib/v4/pkg/liberrors"
	"github.com/google/uuid"

	"github.com/erh/viamrtsp/formatprocessor"

	"github.com/pion/rtp"
	"github.com/pkg/errors"
	goutils "go.viam.com/utils"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/components/camera/rtppassthrough"
	"go.viam.com/rdk/gostream"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/rimage/transform"
)

var (
	family                       = resource.ModelNamespace("erh").WithFamily("viamrtsp")
	ModelAgnostic                = family.WithModel("rtsp")
	ModelH264                    = family.WithModel("rtsp-h264")
	ModelH265                    = family.WithModel("rtsp-h265")
	ModelMJPEG                   = family.WithModel("rtsp-mjpeg")
	Models                       = []resource.Model{ModelAgnostic, ModelH264, ModelH265, ModelMJPEG}
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
		return nil, err
	}
	if conf.IntrinsicParams != nil {
		if err := conf.IntrinsicParams.CheckValid(); err != nil {
			return nil, err
		}
	}
	if conf.DistortionParams != nil {
		if err := conf.DistortionParams.CheckValid(); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

type unitSubscriberFunc func(formatprocessor.Unit) error
type subAndCB struct {
	cb  unitSubscriberFunc
	sub *rtppassthrough.StreamSubscription
}

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

	rtpPassthrough bool
	currentCodec   atomic.Int64

	subsMu       sync.RWMutex
	subAndCBByID map[rtppassthrough.SubscriptionID]subAndCB
}

// Close closes the camera. It always returns nil, but because of Close() interface, it needs to return an error.
func (rc *rtspCamera) Close(ctx context.Context) error {
	rc.cancelFunc()
	rc.unsubscribeAll()
	rc.activeBackgroundWorkers.Wait()
	rc.closeConnection()
	return nil
}

// clientReconnectBackgroundWorker checks every 5 sec to see if the client is connected to the server, and reconnects if not.
func (rc *rtspCamera) clientReconnectBackgroundWorker(codecInfo videoCodec) {
	rc.activeBackgroundWorkers.Add(1)
	goutils.ManagedGo(func() {
		for goutils.SelectContextOrWait(rc.cancelCtx, 5*time.Second) {
			badState := false

			// use an OPTIONS request to see if the server is still responding to requests
			if rc.client == nil {
				badState = true
			} else {
				res, err := rc.client.Options(rc.u)
				rc.logger.Debugf("Options reponse: %s, err: %s", res, err)
				// Nick S:
				// This error happens all the time on hardware we need to support & does not affect
				// the performance of camera streaming. As a result, we ignore this error specifically
				_, isErrClientInvalidState := err.(liberrors.ErrClientInvalidState)
				if err != nil && !isErrClientInvalidState {
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
	default:
		return errors.Errorf("codec not supported %v", codecInfo)
	}

	if _, err := rc.client.Play(nil); err != nil {
		return err
	}
	clientSuccessful = true
	rc.currentCodec.Store(int64(codecInfo))
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
	rc.rawDecoder, err = newH264Decoder()
	if err != nil {
		return errors.Wrap(err, "creating H264 raw decoder")
	}

	// if SPS and PPS are present into the SDP, send them to the decoder
	if f.SPS != nil {
		rc.rawDecoder.decode(f.SPS) // nolint:errcheck
	} else {
		rc.logger.Warn("no SPS found in H264 format")
	}
	if f.PPS != nil {
		rc.rawDecoder.decode(f.PPS) // nolint:errcheck
	} else {
		rc.logger.Warn("no PPS found in H264 format")
	}

	storeImage := func(pkt *rtp.Packet) {
		au, err := rtpDec.Decode(pkt)
		if err != nil {
			if err != rtph264.ErrNonStartingPacketAndNoPrevious && err != rtph264.ErrMorePacketsNeeded {
				rc.logger.Errorf("error decoding(1) h264 rstp stream err: %s", err.Error())
			}
			return
		}

		for _, nalu := range au {
			// convert NALUs into RGBA frames
			image, err := rc.rawDecoder.decode(nalu)
			if err != nil {
				rc.logger.Errorf("error decoding(2) h264 rtsp stream  %s", err.Error())
				return
			}
			if image != nil {
				rc.latestFrame.Store(&image)
			}
		}
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
			if len(rc.subAndCBByID) == 0 {
				return
			}

			// Publish the newly received packet Unit to all subscribers
			for _, subAndCB := range rc.subAndCBByID {
				if err := subAndCB.sub.Publish(func() error { return subAndCB.cb(u) }); err != nil {
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

	rc.rawDecoder, err = newH265Decoder()
	if err != nil {
		return errors.Wrap(err, "creating H265 raw decoder")
	}

	// For H.265, handle VPS, SPS, and PPS
	if f.VPS != nil {
		rc.rawDecoder.decode(f.VPS) // nolint:errcheck
	} else {
		rc.logger.Warn("no VPS found in H265 format")
	}

	if f.SPS != nil {
		rc.rawDecoder.decode(f.SPS) // nolint:errcheck
	} else {
		rc.logger.Warn("no SPS found in H265 format")
	}

	if f.PPS != nil {
		rc.rawDecoder.decode(f.PPS) // nolint:errcheck
	} else {
		rc.logger.Warnf("no PPS found in H265 format")
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
			if err != rtph265.ErrNonStartingPacketAndNoPrevious && err != rtph265.ErrMorePacketsNeeded {
				rc.logger.Errorf("error decoding(1) h265 rstp stream err: %s", err.Error())
			}
			return
		}

		for _, nalu := range au {
			lastImage, err := rc.rawDecoder.decode(nalu)
			if err != nil {
				rc.logger.Errorf("error decoding(2) h265 rtsp stream err: %s", err.Error())
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
func (rc *rtspCamera) SubscribeRTP(ctx context.Context, bufferSize int, packetsCB rtppassthrough.PacketCallback) (rtppassthrough.SubscriptionID, error) {
	if err := rc.validateSupportsPassthrough(); err != nil {
		rc.logger.Debug(err.Error())
		return uuid.Nil, ErrH264PassthroughNotEnabled
	}

	sub, err := rtppassthrough.NewStreamSubscription(bufferSize, func(err error) {
		rc.logger.Errorf("stream subscription hit err: %s", err)
	})
	if err != nil {
		return uuid.Nil, err
	}
	webrtcPayloadMaxSize := 1188 // 1200 - 12 (RTP header)
	encoder := &rtph264.Encoder{
		PayloadType:    96,
		PayloadMaxSize: webrtcPayloadMaxSize,
	}

	if err := encoder.Init(); err != nil {
		return uuid.Nil, err
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
	unitSubscriberFunc := func(u formatprocessor.Unit) error {
		tunit, ok := u.(*formatprocessor.H264)
		if !ok {
			return errors.New("(*unit.H264) type conversion error")
		}

		// If we have no AUs we can't encode packets.
		if tunit.AU == nil {
			return nil
		}

		if !firstReceived {
			firstReceived = true
		} else if tunit.PTS < lastPTS {
			err := errors.New("WebRTC doesn't support H264 streams with B-frames")
			rc.logger.Error(err.Error())
			return err
		}
		lastPTS = tunit.PTS

		pkts, err := encoder.Encode(tunit.AU)
		if err != nil {
			// If there is an Encode error we just drop the packets.
			return nil //nolint:nilerr
		}

		if len(pkts) == 0 {
			// If no packets can be encoded from the AU, there is no need to call the subscriber's callback.
			return nil
		}

		for _, pkt := range pkts {
			pkt.Timestamp += tunit.RTPPackets[0].Timestamp
		}

		return packetsCB(pkts)
	}

	rc.subsMu.Lock()
	defer rc.subsMu.Unlock()

	rc.subAndCBByID[sub.ID()] = subAndCB{cb: unitSubscriberFunc, sub: sub}
	sub.Start()
	return sub.ID(), nil
}

// Unsubscribe deregisters the StreamSubscription's callback.
func (rc *rtspCamera) Unsubscribe(ctx context.Context, id rtppassthrough.SubscriptionID) error {
	rc.subsMu.Lock()
	defer rc.subsMu.Unlock()
	subAndCB, ok := rc.subAndCBByID[id]
	if !ok {
		return errors.New("id not found")
	}
	subAndCB.sub.Close()
	delete(rc.subAndCBByID, id)
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
	rc := &rtspCamera{
		model:          conf.Model,
		u:              u,
		rtpPassthrough: newConf.RTPPassthrough,
		subAndCBByID:   make(map[rtppassthrough.SubscriptionID]subAndCB),
		logger:         logger,
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
	reader := gostream.VideoReaderFunc(func(ctx context.Context) (image.Image, func(), error) {
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
	for id, subAndCB := range rc.subAndCBByID {
		subAndCB.sub.Close()
		delete(rc.subAndCBByID, id)
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
