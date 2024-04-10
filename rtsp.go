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

	"github.com/pion/rtp"
	"github.com/pkg/errors"
	goutils "go.viam.com/utils"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/gostream"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/rimage/transform"
)

var (
	family        = resource.ModelNamespace("erh").WithFamily("viamrtsp")
	ModelAgnostic = family.WithModel("rtsp")
	ModelH264     = family.WithModel("rtsp-h264")
	ModelH265     = family.WithModel("rtsp-h265")
	ModelMJPEG    = family.WithModel("rtsp-mjpeg")
	Models        = []resource.Model{ModelAgnostic, ModelH264, ModelH265, ModelMJPEG}
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
	IntrinsicParams  *transform.PinholeCameraIntrinsics `json:"intrinsic_parameters,omitempty"`
	DistortionParams *transform.BrownConrady            `json:"distortion_parameters,omitempty"`
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

// rtspCamera contains the rtsp client, and the reader function that fulfills the camera interface.
type rtspCamera struct {
	gostream.VideoReader
	u *base.URL

	client     *gortsplib.Client
	rawDecoder *decoder

	cancelCtx  context.Context
	cancelFunc context.CancelFunc

	activeBackgroundWorkers sync.WaitGroup

	latestFrame atomic.Pointer[image.Image]

	logger logging.Logger
}

// Close closes the camera. It always returns nil, but because of Close() interface, it needs to return an error.
func (rc *rtspCamera) Close(ctx context.Context) error {
	rc.cancelFunc()
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
	if rc.rawDecoder != nil {
		rc.rawDecoder.close()
		rc.rawDecoder = nil
	}
}

// reconnectClient reconnects the RTSP client to the streaming server by closing the old one and starting a new one.
func (rc *rtspCamera) reconnectClient(codecInfo videoCodec) error {
	if rc == nil {
		return errors.New("rtspCamera is nil")
	}

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
		codecInfo = getAvailableCodec(tracks)
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

	_, err = rc.client.Setup(session.BaseURL, media, 0, 0)
	if err != nil {
		return errors.Wrapf(err, "when calling RTSP Setup on %s for H264", session.BaseURL)
	}

	rc.client.OnPacketRTP(media, f, onPacketRTP)

	return nil
}

// initH265 initializes the H265 decoder and sets up the client to receive H265 packets.
func (rc *rtspCamera) initH265(session *description.Session) (err error) {
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
	rtspCam := &rtspCamera{
		u:      u,
		logger: logger,
	}
	codecInfo, err := modelToCodec(conf.Model)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	err = rtspCam.reconnectClient(codecInfo)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	cancelCtx, cancel := context.WithCancel(context.Background())
	reader := gostream.VideoReaderFunc(func(ctx context.Context) (image.Image, func(), error) {
		latest := rtspCam.latestFrame.Load()
		if latest == nil {
			return nil, func() {}, errors.New("no frame yet")
		}
		return *latest, func() {}, nil
	})
	rtspCam.VideoReader = reader
	rtspCam.cancelCtx = cancelCtx
	rtspCam.cancelFunc = cancel
	cameraModel := camera.NewPinholeModelWithBrownConradyDistortion(newConf.IntrinsicParams, newConf.DistortionParams)
	rtspCam.clientReconnectBackgroundWorker(codecInfo)
	src, err := camera.NewVideoSourceFromReader(ctx, rtspCam, &cameraModel, camera.ColorStream)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	return camera.FromVideoSource(conf.ResourceName(), src, logger), nil
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
