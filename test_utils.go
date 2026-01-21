package viamrtsp

import (
	"context"
	"encoding/base64"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/bluenviron/mediacommon/pkg/codecs/h264"
	"go.viam.com/rdk/logging"
	"go.viam.com/test"
	"go.viam.com/utils"
)

// NewMockH264ServerHandler creates a new H264 server handler for testing.
func NewMockH264ServerHandler(
	t *testing.T,
	forma *format.H264,
	bURL *base.URL,
	logger logging.Logger,
) (*ServerHandler, func()) {
	stopCtx, stopFunc := context.WithCancel(context.Background())
	//nolint:lll
	h264Base64 := "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"
	h := &ServerHandler{
		media: &description.Media{
			Type:    description.MediaTypeVideo,
			Formats: []format.Format{forma},
		},
		OnSessionCloseFunc: func(_ *gortsplib.ServerHandlerOnSessionCloseCtx, sh *ServerHandler) {
			logger.Debug("OnSessionCloseFunc")
			sh.mu.Lock()
			defer sh.mu.Unlock()
		},
		OnDescribeFunc: func(_ *gortsplib.ServerHandlerOnDescribeCtx, sh *ServerHandler) (*base.Response, *gortsplib.ServerStream, error) {
			logger.Debug("OnDescribeFunc")
			defer logger.Debug("OnDescribeFunc DONE")

			sh.mu.Lock()
			defer sh.mu.Unlock()
			// Only create stream once - reuse for concurrent connections
			if sh.stream == nil {
				d := &description.Session{
					BaseURL: bURL,
					Title:   "123456",
					Medias:  []*description.Media{sh.media},
				}
				sh.stream = gortsplib.NewServerStream(sh.S, d)
				logger.Debug("sh.stream.Description().Medias[0].Formats[0]: %#v ", sh.stream.Description().Medias[0].Formats[0])
			}
			return &base.Response{StatusCode: base.StatusOK}, sh.stream, nil
		},
		OnAnnounceFunc: func(_ *gortsplib.ServerHandlerOnAnnounceCtx, _ *ServerHandler) (*base.Response, error) {
			logger.Debug("OnAnnounceFunc")
			t.Log("should not be called")
			t.FailNow()
			return nil, errors.New("should not be called")
		},
		OnSetupFunc: func(_ *gortsplib.ServerHandlerOnSetupCtx, sh *ServerHandler) (*base.Response, *gortsplib.ServerStream, error) {
			logger.Debug("OnSetupFunc")

			sh.mu.Lock()
			defer sh.mu.Unlock()

			// In tests we expect DESCRIBE to have initialized the stream.
			// If not, fail loudly so races are caught early.
			if sh.stream == nil {
				return &base.Response{StatusCode: base.StatusBadRequest}, nil, errors.New("stream not initialized before SETUP")
			}

			return &base.Response{StatusCode: base.StatusOK}, sh.stream, nil
		},
		// This will play an H264 video which only has frames which are red squares
		// This is so that the result of GetImage is deterministic
		OnPlayFunc: func(_ *gortsplib.ServerHandlerOnPlayCtx, sh *ServerHandler) (*base.Response, error) {
			logger.Debug("OnPlayFunc")
			// Only start the packet writer goroutine once, even with concurrent connections
			sh.playOnce.Do(func() {
				sh.wg.Add(1)
				utils.ManagedGo(func() {
					rtpEnc, err := forma.CreateEncoder()
					if err != nil {
						t.Log(err.Error())
						t.FailNow()
					}

					start := time.Now()
					// setup a ticker to sleep between frames
					//nolint:all
					ticker := time.NewTicker(200 * time.Millisecond)
					defer ticker.Stop()
					b, err := base64.StdEncoding.DecodeString(h264Base64)
					test.That(t, err, test.ShouldBeNil)
					aus, err := h264.AnnexBUnmarshal(b)
					test.That(t, err, test.ShouldBeNil)

					// Continuously send packets until the server is stopped
					for range ticker.C {
						if stopCtx.Err() != nil {
							return
						}
						sh.mu.Lock()
						if sh.stream == nil {
							sh.mu.Unlock()
							return
						}
						sh.mu.Unlock()

						pkts, err := rtpEnc.Encode(aus)
						if err != nil {
							t.Log(err.Error())
							t.FailNow()
						}

						// write packets to the server
						for _, pkt := range pkts {
							// #nosec
							pkt.Timestamp = uint32(multiplyAndDivide(int64(time.Since(start)), int64(forma.ClockRate()), int64(time.Second)))

							sh.mu.Lock()
							err = sh.stream.WritePacketRTP(sh.media, pkt)
							sh.mu.Unlock()
							if err != nil {
								logger.Debug(err.Error())
								return
							}
						}
					}
				}, sh.wg.Done)
			})
			return &base.Response{StatusCode: base.StatusOK}, nil
		},
		OnRecordFunc: func(_ *gortsplib.ServerHandlerOnRecordCtx, _ *ServerHandler) (*base.Response, error) {
			logger.Debug("OnRecordFunc")
			t.Log("should not be called")
			t.FailNow()
			return nil, errors.New("should not be called")
		},
	}

	h.S = &gortsplib.Server{
		Handler:     h,
		RTSPAddress: "127.0.0.1:32513",
	}
	return h, func() {
		stopFunc()
		h.S.Close()
		test.That(t, h.S.Wait(), test.ShouldBeError, errors.New("terminated"))
		h.wg.Wait()
	}
}

// ServerHandler is a handler for a fake RTSP server.
type ServerHandler struct {
	S                  *gortsplib.Server
	wg                 sync.WaitGroup
	media              *description.Media
	OnConnOpenFunc     func(*gortsplib.ServerHandlerOnConnOpenCtx, *ServerHandler)
	OnConnCloseFunc    func(*gortsplib.ServerHandlerOnConnCloseCtx, *ServerHandler)
	OnSessionOpenFunc  func(*gortsplib.ServerHandlerOnSessionOpenCtx, *ServerHandler)
	OnSessionCloseFunc func(*gortsplib.ServerHandlerOnSessionCloseCtx, *ServerHandler)
	OnDescribeFunc     func(*gortsplib.ServerHandlerOnDescribeCtx, *ServerHandler) (*base.Response, *gortsplib.ServerStream, error)
	OnAnnounceFunc     func(*gortsplib.ServerHandlerOnAnnounceCtx, *ServerHandler) (*base.Response, error)
	OnSetupFunc        func(*gortsplib.ServerHandlerOnSetupCtx, *ServerHandler) (*base.Response, *gortsplib.ServerStream, error)
	OnPlayFunc         func(*gortsplib.ServerHandlerOnPlayCtx, *ServerHandler) (*base.Response, error)
	OnRecordFunc       func(*gortsplib.ServerHandlerOnRecordCtx, *ServerHandler) (*base.Response, error)
	mu                 sync.Mutex
	stream             *gortsplib.ServerStream
	playOnce           sync.Once
}

// OnConnOpen called when a connection is opened.
func (sh *ServerHandler) OnConnOpen(ctx *gortsplib.ServerHandlerOnConnOpenCtx) {
	if sh.OnConnOpenFunc == nil {
		return
	}
	sh.OnConnOpenFunc(ctx, sh)
}

// OnConnClose called when a connection is closed.
func (sh *ServerHandler) OnConnClose(ctx *gortsplib.ServerHandlerOnConnCloseCtx) {
	if sh.OnConnCloseFunc == nil {
		return
	}
	sh.OnConnCloseFunc(ctx, sh)
}

// OnSessionOpen called when a session is opened.
func (sh *ServerHandler) OnSessionOpen(ctx *gortsplib.ServerHandlerOnSessionOpenCtx) {
	if sh.OnSessionOpenFunc == nil {
		return
	}
	sh.OnSessionOpenFunc(ctx, sh)
}

// OnSessionClose called when a session is closed.
func (sh *ServerHandler) OnSessionClose(ctx *gortsplib.ServerHandlerOnSessionCloseCtx) {
	if sh.OnSessionCloseFunc == nil {
		return
	}
	sh.OnSessionCloseFunc(ctx, sh)
}

// OnDescribe called when receiving a DESCRIBE request.
func (sh *ServerHandler) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (*base.Response, *gortsplib.ServerStream, error) {
	return sh.OnDescribeFunc(ctx, sh)
}

// OnAnnounce called when receiving an ANNOUNCE request.
func (sh *ServerHandler) OnAnnounce(ctx *gortsplib.ServerHandlerOnAnnounceCtx) (*base.Response, error) {
	return sh.OnAnnounceFunc(ctx, sh)
}

// OnSetup called when receiving a SETUP request.
func (sh *ServerHandler) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
	return sh.OnSetupFunc(ctx, sh)
}

// OnPlay called when receiving a PLAY request.
func (sh *ServerHandler) OnPlay(ctx *gortsplib.ServerHandlerOnPlayCtx) (*base.Response, error) {
	return sh.OnPlayFunc(ctx, sh)
}

// OnRecord called when receiving a RECORD request.
func (sh *ServerHandler) OnRecord(ctx *gortsplib.ServerHandlerOnRecordCtx) (*base.Response, error) {
	return sh.OnRecordFunc(ctx, sh)
}

func multiplyAndDivide(v, m, d int64) int64 {
	secs := v / d
	dec := v % d
	return (secs*m + dec*m/d)
}
