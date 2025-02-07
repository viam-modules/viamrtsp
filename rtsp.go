// Package viamrtsp implements RTSP camera support in a Viam module
package viamrtsp

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/bluenviron/mediacommon/pkg/codecs/h265"
	"github.com/erh/viamupnp"
	"github.com/pion/rtp"
	"github.com/viam-modules/viamrtsp/formatprocessor"
	"github.com/viam-modules/viamrtsp/viamonvif"
	"github.com/viam-modules/viamrtsp/viamonvif/device"
	"go.uber.org/zap/zapcore"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/components/camera/rtppassthrough"
	"go.viam.com/rdk/gostream"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/pointcloud"
	"go.viam.com/rdk/resource"
	rutils "go.viam.com/rdk/utils"
	"go.viam.com/utils"
)

const (
	// reconnectIntervalSeconds is the interval in secs to wait before reconnecting the background worker.
	reconnectIntervalSeconds = 5
	// webRTCPayloadMaxSize is the maximum size of a WebRTC RTP payload, calculated as 1200 - 12 (RTP header).
	webRTCPayloadMaxSize = 1188
	// defaultPayloadType is the default payload type for RTP packets.
	defaultPayloadType = 96
	// h264NALUTypeMask is the mask to extract the NALU type from the first byte of an H264 NALU.
	h264NALUTypeMask = 0x1F
	// initialFramePoolSize is the initial size of the frame pool.
	initialFramePoolSize = 5
)

var (
	// Family is the namespace family for the viamrtsp module.
	Family = resource.ModelNamespace("viam").WithFamily("viamrtsp")
	// ModelAgnostic selects the best available codec.
	ModelAgnostic = Family.WithModel("rtsp")
	// ModelH264 uses the h264 codec.
	ModelH264 = Family.WithModel("rtsp-h264")
	// ModelH265 uses the h265 codec.
	ModelH265 = Family.WithModel("rtsp-h265")
	// ModelMJPEG uses the mjpeg codec.
	ModelMJPEG = Family.WithModel("rtsp-mjpeg")
	// ModelMPEG4 uses the mpeg4 codec.
	ModelMPEG4 = Family.WithModel("rtsp-mpeg4")
	// Models is a slice containing the above available models.
	Models = []resource.Model{ModelAgnostic, ModelH264, ModelH265, ModelMJPEG, ModelMPEG4}
	// ErrH264PassthroughNotEnabled is an error indicating H264 passthrough is not enabled.
	ErrH264PassthroughNotEnabled = errors.New("H264 passthrough is not enabled")
)

func getStringFromExtra(extra map[string]interface{}, key string) (string, error) {
	extraVal, ok := extra[key]
	if !ok {
		return "", fmt.Errorf("key not found in 'extra': %s", key)
	}
	extraValStr, ok := extraVal.(string)
	if !ok {
		return "", errors.New("'extra' value must be a string")
	}
	return extraValStr, nil
}

func init() {
	for _, model := range Models {
		resource.RegisterComponent(camera.API, model, resource.Registration[camera.Camera, *Config]{
			Constructor: NewRTSPCamera,
			Discover: func(ctx context.Context, logger logging.Logger, extra map[string]interface{}) (interface{}, error) {
				logger.Debugf("viamrtsp discovery received extra credentials: %v", extra)
				username, err := getStringFromExtra(extra, "username")
				if err != nil {
					logger.Infof("error getting username from extra: %v", err)
				}
				password, err := getStringFromExtra(extra, "password")
				if err != nil {
					logger.Infof("error getting password from extra: %v", err)
				}
				creds := []device.Credentials{{User: username, Pass: password}}
				camInfoList, err := viamonvif.DiscoverCameras(ctx, creds, nil, logger)
				if err != nil {
					return nil, err
				}
				return camInfoList, nil
			},
		})
	}
}

// Config are the config attributes for an RTSP camera model.
type Config struct {
	Address        string               `json:"rtsp_address"`
	RTPPassthrough *bool                `json:"rtp_passthrough"`
	LazyDecode     bool                 `json:"lazy_decode,omitempty"`
	Query          viamupnp.DeviceQuery `json:"query"`
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

	return nil, nil
}

func (conf *Config) parseAndFixAddress(ctx context.Context, logger logging.Logger) (*base.URL, error) {
	u, err := base.ParseURL(conf.Address)
	if err != nil {
		return nil, err
	}

	if u.Hostname() == "UPNP_DISCOVER" {
		host, err := viamupnp.FindHost(ctx, logger, conf.Query)
		if err != nil {
			return nil, err
		}

		p := u.Port()
		if p == "" {
			u.Host = host
		} else {
			u.Host = fmt.Sprintf("%s:%s", host, u.Port())
		}
	}

	return u, nil
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
	resource.AlwaysRebuild
	model resource.Model
	gostream.VideoReader
	u          *base.URL
	lazyDecode bool

	closeMu sync.RWMutex

	auMu       sync.Mutex
	au         [][]byte
	client     *gortsplib.Client
	rawDecoder *decoder

	cancelCtx  context.Context
	cancelFunc context.CancelFunc

	activeBackgroundWorkers sync.WaitGroup

	latestMJPEGBytes atomic.Pointer[[]byte]
	latestFrame      *avFrameWrapper
	// frameSwapMu protects critical sections where frame state changes (e.g. ref counting) need to be atomic
	// with swapping out the latest frame.
	frameSwapMu sync.Mutex
	// We use a pool data structure to amortize the malloc cost of AVFrames and reduce pressure on memory
	// management. We create one pool for the entire lifetime of the RTSP camera. Additionally, frames
	// from the pool may be for a resolution that does not match the current image. The user of the pool
	// is responsible for the underlying frame contents and further initializing it and/or throwing it away.
	avFramePool *framePool

	mimeHandler *mimeHandler

	logger logging.Logger

	rtpPassthrough              bool
	currentCodec                atomic.Int64
	rtpPassthroughCtx           context.Context
	rtpPassthroughCancelCauseFn context.CancelCauseFunc

	subsMu       sync.RWMutex
	bufAndCBByID map[rtppassthrough.SubscriptionID]bufAndCB

	name resource.Name
}

// Close closes the camera. It always returns nil, but because of Close() interface, it needs to return an error.
func (rc *rtspCamera) Close(_ context.Context) error {
	rc.cancelFunc()
	rc.closeMu.Lock()
	defer rc.closeMu.Unlock()
	rc.unsubscribeAll()
	rc.activeBackgroundWorkers.Wait()
	rc.closeConnection()
	rc.avFramePool.close()
	rc.mimeHandler.close()
	return nil
}

// clientReconnectBackgroundWorker checks every reconnectIntervalSeconds to see if the client is connected to the server,
// and reconnects if not.
func (rc *rtspCamera) clientReconnectBackgroundWorker(codecInfo videoCodec) {
	rc.activeBackgroundWorkers.Add(1)
	utils.ManagedGo(func() {
		for utils.SelectContextOrWait(rc.cancelCtx, reconnectIntervalSeconds*time.Second) {
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
				if err := rc.reconnectClientWithFallbackTransports(codecInfo); err != nil {
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
	rc.resetLazyAU([][]byte{})
	rc.currentCodec.Store(0)
	if rc.rawDecoder != nil {
		rc.rawDecoder.close()
		rc.rawDecoder = nil
	}
}

// reconnectClientWithFallbackTransports attempts to setup the RTSP client with the given codec
// using the transports in the order of TCP, UDP, and UDP Multicast. This overrides gortsplib's
// default behavior of trying UDP first.
func (rc *rtspCamera) reconnectClientWithFallbackTransports(codecInfo videoCodec) error {
	// Define the transport preferences in order.
	transportTCP := gortsplib.TransportTCP
	transportUDP := gortsplib.TransportUDP
	transportUDPMulticast := gortsplib.TransportUDPMulticast
	transports := []*gortsplib.Transport{
		&transportTCP,
		&transportUDP,
		&transportUDPMulticast,
	}
	// Try to reconnect with each transport in the order defined above.
	// If all attempts fail, return the last error.
	var lastErr error
	for _, transport := range transports {
		if err := rc.reconnectClient(codecInfo, transport); err != nil {
			rc.logger.Warnf("cannot reconnect to rtsp server using transport %s, err: %s", transport.String(), err.Error())
			lastErr = err
			continue
		}
		rc.logger.Debugf("successfully reconnected to rtsp server url: %s", rc.u)
		return nil
	}
	return fmt.Errorf("all attempts to reconnect to rtsp server failed: %w", lastErr)
}

// reconnectClient reconnects the RTSP client to the streaming server by closing the old one and starting a new one.
func (rc *rtspCamera) reconnectClient(codecInfo videoCodec, transport *gortsplib.Transport) error {
	rc.logger.Warnf("reconnectClient called with codec: %s and transport: %s", codecInfo, transport.String())

	rc.closeConnection()

	// replace the client with a new one, but close it if setup is not successful
	rc.client = &gortsplib.Client{
		Transport: transport,
	}
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
		return fmt.Errorf("when calling RTSP START on Scheme: %s, Host: %s, Error: %w", rc.u.Scheme, rc.u.Host, err)
	}

	var clientSuccessful bool
	defer func() {
		if !clientSuccessful {
			rc.closeConnection()
		}
	}()

	session, _, err := rc.client.Describe(rc.u)
	if err != nil {
		return fmt.Errorf("when calling RTSP DESCRIBE on %s: %w", rc.u, err)
	}
	rc.logger.Debugf("Session media info: %+v", session)
	for _, media := range session.Medias {
		for i, format := range media.Formats {
			rc.logger.Debugf("Media %d format: %s", i+1, format.Codec())
			rc.logger.Debugf("Format clock rate: %d", format.ClockRate())
			rc.logger.Debugf("Format payload type: %d", format.PayloadType())
			rc.logger.Debugf("Format RTPMap: %s", format.RTPMap())
			rc.logger.Debugf("Format FMTP: %+v", format.FMTP())
		}
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
	case MPEG4:
		rc.logger.Info("setting up MPEG4 decoder")
		if err := rc.initMPEG4(session); err != nil {
			return err
		}
	case Unknown, Agnostic:
		return fmt.Errorf("codecInfo was '%s' after getting session info. Codec could not be determined", codecInfo.String())
	default:
		return fmt.Errorf("codec not supported: '%s'", codecInfo.String())
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

func (rc *rtspCamera) consumeLazyAU() {
	rc.auMu.Lock()
	defer rc.auMu.Unlock()
	if len(rc.au) > 0 {
		codec := videoCodec(rc.currentCodec.Load())
		switch codec {
		case H264:
			rc.storeH264Frame(rc.au)
		case H265:
			for _, au := range rc.au {
				// h265 AUs are already packed into a single frame
				// before they were added to rc.au
				rc.storeH265Frame(au)
			}
		case Unknown:
		case Agnostic:
		case MJPEG:
		case MPEG4:
			fallthrough
		default:
			rc.logger.Infof("consumeLazyAU: called with unexpected codec: %s, int: %d", codec, codec)
		}

		rc.au = nil
	}
}

func (rc *rtspCamera) resetLazyAU(au [][]byte) {
	rc.auMu.Lock()
	defer rc.auMu.Unlock()
	rc.au = au
}

func (rc *rtspCamera) appendLazyAU(au [][]byte) {
	rc.auMu.Lock()
	defer rc.auMu.Unlock()
	rc.au = append(rc.au, au...)
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
		return fmt.Errorf("creating H264 RTP decoder: %w", err)
	}

	// setup H264 -> raw frames decoder
	rc.rawDecoder, err = newH264Decoder(rc.avFramePool, rc.logger)
	if err != nil {
		return fmt.Errorf("creating H264 raw decoder: %w", err)
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
			rc.storeH264Frame(au)
			return
		}

		if rc.lazyDecode {
			if h264.IDRPresent(au) {
				rc.resetLazyAU(au)
			} else {
				rc.appendLazyAU(au)
			}
		} else {
			rc.storeH264Frame(au)
		}
	}

	onPacketRTP := func(pkt *rtp.Packet) {
		storeImage(pkt)
	}

	if rc.rtpPassthrough {
		fp, err := formatprocessor.New(webRTCPayloadMaxSize, f, true)
		if err != nil {
			return fmt.Errorf("unable to create new h264 rtp formatprocessor: %w", err)
		}

		publishToWebRTC := func(pkt *rtp.Packet) {
			pts, ok := rc.client.PacketPTS2(media, pkt)
			if !ok {
				return
			}
			ntp := time.Now()
			u, err := fp.ProcessRTPPacket(pkt, ntp, time.Duration(pts), true)
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
		return fmt.Errorf("when calling RTSP Setup on %s for H264: %w", session.BaseURL, err)
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
		return fmt.Errorf("creating H265 RTP decoder: %w", err)
	}

	rc.rawDecoder, err = newH265Decoder(rc.avFramePool, rc.logger)
	if err != nil {
		return fmt.Errorf("creating H265 raw decoder: %w", err)
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
		return fmt.Errorf("when calling RTSP Setup on %s for H265: %w", session.BaseURL, err)
	}

	// On packet retreival, turn it into an image, and store it in shared memory
	rc.client.OnPacketRTP(media, f, func(pkt *rtp.Packet) {
		// Extract access units from RTP packets
		au, err := rtpDec.Decode(pkt)
		if err != nil {
			if !errors.Is(err, rtph265.ErrNonStartingPacketAndNoPrevious) && !errors.Is(err, rtph265.ErrMorePacketsNeeded) {
				rc.logger.Debugw("error decoding(1) h265 rstp stream", "err", err.Error())
			}
			return
		}

		packedAU := packH265AUIntoNALU(au, rc.logger)
		if rc.lazyDecode {
			if h265.IsRandomAccess(au) {
				rc.resetLazyAU([][]byte{packedAU})
			} else {
				rc.appendLazyAU([][]byte{packedAU})
			}
		} else {
			rc.storeH265Frame(packedAU)
		}
	})

	return nil
}

func packH265AUIntoNALU(au [][]byte, logger logging.Logger) []byte {
	// If the AU has more than one NALU, compact them into a single payload with NALUs separated
	// in AnnexB format. This is necessary because the H.265 decoder expects all NALUs for a frame
	// to be in a single payload rather than chunked across multiple decode calls.
	packedNALU := []byte{}
	firstNALU := true
	for _, nalu := range au {
		if len(nalu) == 0 {
			logger.Warn("empty NALU found in H265 AU, skipping NALU")
			continue
		}
		// Add start code prefix to all NALUs except the first non-empty one.
		if !firstNALU {
			packedNALU = append(packedNALU, H2645StartCode()...)
		}
		packedNALU = append(packedNALU, nalu...)
		firstNALU = false
	}
	return packedNALU
}

func (rc *rtspCamera) storeH265Frame(nalu []byte) {
	if len(nalu) == 0 {
		rc.logger.Warn("no NALUs found in H265 AU, skipping packet")
		return
	}

	frame, err := rc.rawDecoder.decode(nalu)
	if err != nil {
		rc.logger.Debugw("error decoding(2) h265 rtsp stream", "err", err.Error())
		return
	}

	if frame != nil {
		rc.handleLatestFrame(frame)
	}
}

// initMJPEG initializes the MJPEG decoder and sets up the client to receive JPEG frames.
func (rc *rtspCamera) initMJPEG(session *description.Session) error {
	if rc.rtpPassthrough {
		rc.logger.Warn("rtp_passthrough is only supported for H264 codec. rtp_passthrough features disabled due to MJPEG RTSP track")
	}
	if rc.lazyDecode {
		rc.logger.Warn("lazy_decode is currently only supported for H264 codec. lazy_decode features disabled due to MJPEG RTSP track")
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
		return fmt.Errorf("creating MJPEG RTP decoder: %w", err)
	}

	_, err = rc.client.Setup(session.BaseURL, media, 0, 0)
	if err != nil {
		return fmt.Errorf("when calling RTSP Setup on %s for MJPEG: %w", session.BaseURL, err)
	}

	rc.client.OnPacketRTP(media, f, func(pkt *rtp.Packet) {
		frame, err := mjpegDecoder.Decode(pkt)
		if err != nil {
			return
		}

		rc.latestMJPEGBytes.Store(&frame)
	})

	return nil
}

// initMPEG4 initializes the MPEG4 decoder and sets up the client to receive MPEG4 packets.
func (rc *rtspCamera) initMPEG4(session *description.Session) error {
	if rc.rtpPassthrough {
		rc.logger.Warn("rtp_passthrough is only supported for H264 codec. rtp_passthrough features disabled due to MPEG4 RTSP track")
	}

	if rc.lazyDecode {
		rc.logger.Warn("lazy_decode is currently only supported for H264 codec. lazy_decode features disabled due to MPEG4 RTSP track")
	}

	var f *format.MPEG4Video
	media := session.FindFormat(&f)
	if media == nil {
		rc.logger.Warn("No MPEG4 track found")
		for _, x := range session.Medias {
			rc.logger.Warnf("\t %v", x)
		}
		return errors.New("MPEG4 track not found")
	}

	mpeg4Decoder, err := f.CreateDecoder()
	if err != nil {
		return fmt.Errorf("creating MPEG4 RTP decoder: %w", err)
	}

	// Initialize the rawDecoder with MPEG4 config data
	if f.Config != nil {
		// Prepend MPEG4 Visual Object Sequence (VOS) and Video Object (VO) start codes
		vosStart := []byte{0x00, 0x00, 0x01, 0xB0}
		voStart := []byte{0x00, 0x00, 0x01, 0xB5}
		extraData := append(vosStart, voStart...)
		extraData = append(extraData, f.Config...)

		rc.rawDecoder, err = newMPEG4Decoder(rc.avFramePool, rc.logger, extraData)
		if err != nil {
			return fmt.Errorf("creating MPEG4 raw decoder: %w", err)
		}
	} else {
		rc.rawDecoder, err = newMPEG4Decoder(rc.avFramePool, rc.logger, nil)
		if err != nil {
			return fmt.Errorf("creating MPEG4 raw decoder: %w", err)
		}
	}

	_, err = rc.client.Setup(session.BaseURL, media, 0, 0)
	if err != nil {
		return fmt.Errorf("when calling RTSP Setup on %s for MPEG4: %w", session.BaseURL, err)
	}

	rc.client.OnPacketRTP(media, f, func(pkt *rtp.Packet) {
		frame, err := mpeg4Decoder.Decode(pkt)
		if err != nil {
			return
		}

		if frame != nil {
			if decodedFrame, err := rc.rawDecoder.decode(frame); err == nil && decodedFrame != nil {
				rc.handleLatestFrame(decodedFrame)
			}
		}
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
		PayloadType:    defaultPayloadType,
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

// NewRTSPCamera creates a new rtsp camera from the config, that has to have a viamrtsp.Config.
func NewRTSPCamera(ctx context.Context, _ resource.Dependencies, conf resource.Config, logger logging.Logger) (camera.Camera, error) {
	if logger.Level() != zapcore.DebugLevel {
		logger.Info("suppressing non fatal libav errors / warnings due to false positives. to unsuppress, set module log_level to 'debug'")
		SetLibAVLogLevelFatal()
	}

	newConf, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	u, err := newConf.parseAndFixAddress(ctx, logger)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	framePool := newFramePool(initialFramePoolSize, logger)

	mimeHandler := newMimeHandler(logger)

	rtpPassthrough := true
	if newConf.RTPPassthrough != nil {
		rtpPassthrough = *newConf.RTPPassthrough
	}

	rtpPassthroughCtx, rtpPassthroughCancelCauseFn := context.WithCancelCause(context.Background())
	rc := &rtspCamera{
		model:                       conf.Model,
		lazyDecode:                  newConf.LazyDecode,
		u:                           u,
		name:                        conf.ResourceName(),
		rtpPassthrough:              rtpPassthrough,
		bufAndCBByID:                make(map[rtppassthrough.SubscriptionID]bufAndCB),
		rtpPassthroughCtx:           rtpPassthroughCtx,
		rtpPassthroughCancelCauseFn: rtpPassthroughCancelCauseFn,
		avFramePool:                 framePool,
		mimeHandler:                 mimeHandler,
		logger:                      logger,
	}
	codecInfo, err := modelToCodec(conf.Model)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	err = rc.reconnectClientWithFallbackTransports(codecInfo)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	cancelCtx, cancel := context.WithCancel(context.Background())
	rc.cancelCtx = cancelCtx
	rc.cancelFunc = cancel
	rc.clientReconnectBackgroundWorker(codecInfo)

	return rc, nil
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
		return fmt.Errorf("rtp_passthrough was determined to not be supported at runtime due to %w", err)
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
	case ModelMPEG4:
		return MPEG4, nil
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
	var mpeg4 *format.MPEG4Video

	// List of formats/codecs in priority order
	codecFormats := []codecFormat{
		{&h264, H264},
		{&h265, H265},
		{&mjpeg, MJPEG},
		{&mpeg4, MPEG4},
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
			nalu, nalusCompacted := compactH264SPSAndPPSAndIDR(au[naluIndex:])
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

func compactH264SPSAndPPSAndIDR(au [][]byte) ([]byte, int) {
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

// H2645StartCode is the start code byte sequence for H264/H265 NALs.
func H2645StartCode() []byte {
	return []uint8{0x00, 0x00, 0x00, 0x01}
}

func (rc *rtspCamera) decodeAndStore(nalu []byte) error {
	frame, err := rc.rawDecoder.decode(nalu)
	recoverableErr := &recoverableError{}
	if errors.As(err, &recoverableErr) {
		return nil
	}
	if err != nil {
		return err
	}
	if frame != nil {
		rc.handleLatestFrame(frame)
	}
	return nil
}

// handleLatestFrame sets the new latest frame, and cleans up
// the previous frame by trying to put it back in the pool. It might not make
// it back into the pool immediately or at all depending on its state.
func (rc *rtspCamera) handleLatestFrame(newFrame *avFrameWrapper) {
	rc.frameSwapMu.Lock()
	defer rc.frameSwapMu.Unlock()

	prevFrame := rc.latestFrame
	if prevFrame != nil {
		if refCount := prevFrame.decrementRefs(); refCount == 0 {
			rc.avFramePool.put(prevFrame)
		}
	}
	newFrame.incrementRefs()
	rc.latestFrame = newFrame
}

func naluType(nalu []byte) h264.NALUType {
	return h264.NALUType(nalu[0] & h264NALUTypeMask)
}

func isCompactableH264(nalu []byte) bool {
	typ := naluType(nalu)
	return typ == h264.NALUTypeSPS || typ == h264.NALUTypePPS || typ == h264.NALUTypeIDR
}

// Image returns the latest frame as JPEG bytes.
func (rc *rtspCamera) Image(_ context.Context, mimeType string, _ map[string]interface{}) ([]byte, camera.ImageMetadata, error) {
	rc.closeMu.RLock()
	defer rc.closeMu.RUnlock()
	start := time.Now()
	defer func() {
		rc.logger.Infof("codec: %s, lazy: %t, time: %s", videoCodec(rc.currentCodec.Load()), rc.lazyDecode, time.Since(start))
	}()
	if err := rc.cancelCtx.Err(); err != nil {
		return nil, camera.ImageMetadata{}, err
	}
	if videoCodec(rc.currentCodec.Load()) == MJPEG {
		mjpegBytes := rc.latestMJPEGBytes.Load()
		if mjpegBytes == nil {
			return nil, camera.ImageMetadata{}, errors.New("no frame yet")
		}
		return *mjpegBytes, camera.ImageMetadata{
			MimeType: rutils.MimeTypeJPEG,
		}, nil
	}
	return rc.getAndConvertFrame(mimeType)
}

func (rc *rtspCamera) getAndConvertFrame(mimeType string) ([]byte, camera.ImageMetadata, error) {
	rc.consumeLazyAU()
	rc.frameSwapMu.Lock()
	defer rc.frameSwapMu.Unlock()
	if rc.latestFrame == nil {
		return nil, camera.ImageMetadata{}, errors.New("no frame yet")
	}
	currentFrame := rc.latestFrame
	currentFrame.incrementRefs()
	var bytes []byte
	var metadata camera.ImageMetadata
	var err error
	switch mimeType {
	case rutils.MimeTypeJPEG, rutils.MimeTypeJPEG + "+" + rutils.MimeTypeSuffixLazy:
		bytes, metadata, err = rc.mimeHandler.convertJPEG(currentFrame.frame)
	case mimeTypeYUYV, mimeTypeYUYV + "+" + rutils.MimeTypeSuffixLazy:
		bytes, metadata, err = rc.mimeHandler.convertYUYV(currentFrame.frame)
	case rutils.MimeTypeRawRGBA, rutils.MimeTypeRawRGBA + "+" + rutils.MimeTypeSuffixLazy:
		bytes, metadata, err = rc.mimeHandler.convertRGBA(currentFrame.frame)
	default:
		rc.logger.Debugf("unsupported mime type: %s, defaulting to JPEG", mimeType)
		bytes, metadata, err = rc.mimeHandler.convertJPEG(currentFrame.frame)
	}
	if refCount := currentFrame.decrementRefs(); refCount == 0 {
		rc.avFramePool.put(currentFrame)
	}
	if err != nil {
		return nil, camera.ImageMetadata{}, err
	}
	return bytes, metadata, err
}

func (rc *rtspCamera) Properties(_ context.Context) (camera.Properties, error) {
	return camera.Properties{
		SupportsPCD: false,
		MimeTypes:   []string{rutils.MimeTypeJPEG},
	}, nil
}

func (rc *rtspCamera) Name() resource.Name {
	return rc.name
}

// Unimplemented camera interface methods.
func (rc *rtspCamera) Stream(_ context.Context, _ ...gostream.ErrorHandler) (gostream.VideoStream, error) {
	return nil, errors.New("stream not implemented")
}

func (rc *rtspCamera) Images(_ context.Context) ([]camera.NamedImage, resource.ResponseMetadata, error) {
	return nil, resource.ResponseMetadata{}, errors.New("not implemented")
}

func (rc *rtspCamera) NextPointCloud(_ context.Context) (pointcloud.PointCloud, error) {
	return nil, errors.New("not implemented")
}

func (rc *rtspCamera) DoCommand(_ context.Context, _ map[string]interface{}) (map[string]interface{}, error) {
	return nil, errors.New("not implemented")
}
