package viamrtsp

import (
	"context"
	"image"
	"testing"
	"time"

	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/erh/viamupnp"
	"github.com/koron/go-ssdp"
	"github.com/pion/rtp"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/components/camera/rtppassthrough"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	rutils "go.viam.com/rdk/utils"
	"go.viam.com/test"
)

func TestRTSPCamera(t *testing.T) {
	SetLibAVLogLevelFatal()
	logger := logging.NewTestLogger(t)
	bURL, err := base.ParseURL("rtsp://127.0.0.1:32512")
	test.That(t, err, test.ShouldBeNil)

	t.Run("H264", func(t *testing.T) {
		forma := &format.H264{
			PayloadTyp:        96,
			PacketizationMode: 1,
			SPS: []uint8{
				0x67, 0x64, 0x00, 0x15, 0xac, 0xb2, 0x03, 0xc1,
				0x1f, 0xd6, 0x02, 0xdc, 0x08, 0x08, 0x16, 0x94,
				0x00, 0x00, 0x03, 0x00, 0x04, 0x00, 0x00, 0x03,
				0x00, 0xf0, 0x3c, 0x58, 0xb9, 0x20,
			},
			PPS: []uint8{0x68, 0xeb, 0xc3, 0xcb, 0x22, 0xc0},
		}
		t.Run("init", func(t *testing.T) {
			h, closeFunc := NewMockH264ServerHandler(t, forma, bURL, logger)
			test.That(t, h.S.Start(), test.ShouldBeNil)
			timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), time.Second*10)
			defer timeoutCancel()
			config := resource.NewEmptyConfig(camera.Named("foo"), ModelAgnostic)
			config.ConvertedAttributes = &Config{Address: "rtsp://" + h.S.RTSPAddress + "/stream1"}
			rtspCam, err := NewRTSPCamera(timeoutCtx, nil, config, logger)
			test.That(t, err, test.ShouldBeNil)
			defer func() { test.That(t, rtspCam.Close(context.Background()), test.ShouldBeNil) }()
			test.That(t, rtspCam.Name().Name, test.ShouldEqual, "foo")
			closeFunc()
		})

		t.Run("GetImage", func(t *testing.T) {
			h, closeFunc := NewMockH264ServerHandler(t, forma, bURL, logger)
			defer closeFunc()
			test.That(t, h.S.Start(), test.ShouldBeNil)
			timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), time.Second*10)
			defer timeoutCancel()
			config := resource.NewEmptyConfig(camera.Named("foo"), ModelAgnostic)
			config.ConvertedAttributes = &Config{Address: "rtsp://" + h.S.RTSPAddress + "/stream1"}
			rtspCam, err := NewRTSPCamera(timeoutCtx, nil, config, logger)
			test.That(t, err, test.ShouldBeNil)
			defer func() { test.That(t, rtspCam.Close(context.Background()), test.ShouldBeNil) }()
			imageTimeoutCtx, imageTimeoutCancel := context.WithTimeout(context.Background(), time.Second*10)
			defer imageTimeoutCancel()
			var im image.Image
			for imageTimeoutCtx.Err() == nil {
				img, err := camera.DecodeImageFromCamera(imageTimeoutCtx, rtspCam, nil, nil)
				if err != nil {
					continue
				}
				if img != nil {
					im = img
					break
				}
			}
			test.That(t, imageTimeoutCtx.Err(), test.ShouldBeNil)
			test.That(t, im.Bounds(), test.ShouldResemble, image.Rect(0, 0, 480, 270))
		})

		t.Run("AvFramePool", func(t *testing.T) {
			h, closeFunc := NewMockH264ServerHandler(t, forma, bURL, logger)
			defer closeFunc()
			test.That(t, h.S.Start(), test.ShouldBeNil)
			timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), time.Second*10)
			defer timeoutCancel()
			config := resource.NewEmptyConfig(camera.Named("foo"), ModelAgnostic)
			config.ConvertedAttributes = &Config{Address: "rtsp://" + h.S.RTSPAddress + "/stream1"}
			rtspCam, err := NewRTSPCamera(timeoutCtx, nil, config, logger)
			test.That(t, err, test.ShouldBeNil)
			imageTimeoutCtx, imageTimeoutCancel := context.WithTimeout(context.Background(), time.Second*10)
			defer imageTimeoutCancel()
			// Fetch images while allowing the pool to grow and shrink as needed.
			imgCount := 0
			for imageTimeoutCtx.Err() == nil {
				time.Sleep(100 * time.Millisecond)
				img, err := camera.DecodeImageFromCamera(imageTimeoutCtx, rtspCam, nil, nil)
				if err != nil {
					continue
				}
				if img != nil {
					imgCount++
				}
				if imgCount == 30 {
					break
				}
			}
			// Test that we put as many frames as we received back into the pool.
			rtspCam.Close(timeoutCtx)
			test.That(t, imageTimeoutCtx.Err(), test.ShouldBeNil)
			totalPoolFramesSeen := rtspCam.(*rtspCamera).avFramePool.newCount + rtspCam.(*rtspCamera).avFramePool.getCount
			test.That(t, rtspCam.(*rtspCamera).avFramePool.putCount, test.ShouldEqual, totalPoolFramesSeen)
		})

		t.Run("SubscribeRTP", func(t *testing.T) {
			t.Run("RTPPassthrough config variations", func(t *testing.T) {
				cases := []struct {
					name                 string
					rtpPassthrough       *bool
					expectSubscribeError error
				}{
					{"when RTPPassthrough = true", func() *bool { b := true; return &b }(), nil},
					{"when RTPPassthrough = false", func() *bool { b := false; return &b }(), ErrH264PassthroughNotEnabled},
					{"when RTPPassthrough is not specified", nil, nil},
				}

				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						// Server and camera setup
						h, closeFunc := NewMockH264ServerHandler(t, forma, bURL, logger)
						defer closeFunc()
						test.That(t, h.S.Start(), test.ShouldBeNil)

						timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 10*time.Second)
						defer timeoutCancel()

						config := resource.NewEmptyConfig(camera.Named("foo"), ModelAgnostic)
						config.ConvertedAttributes = &Config{
							Address:        "rtsp://" + h.S.RTSPAddress + "/stream1",
							RTPPassthrough: tc.rtpPassthrough,
						}

						rtspCam, err := NewRTSPCamera(timeoutCtx, nil, config, logger)
						test.That(t, err, test.ShouldBeNil)
						defer func() { test.That(t, rtspCam.Close(context.Background()), test.ShouldBeNil) }()

						vcs, ok := rtspCam.(rtppassthrough.Source)
						test.That(t, ok, test.ShouldBeTrue)

						// Subscription and packet tests
						if tc.expectSubscribeError == nil {
							cancelCtx, cancel := context.WithCancel(context.Background())
							sub, err := vcs.SubscribeRTP(timeoutCtx, 512, func(pkts []*rtp.Packet) {
								if len(pkts) > 0 {
									logger.Info("got packets")
									cancel()
								}
							})
							test.That(t, err, test.ShouldBeNil)
							defer func() {
								err := vcs.Unsubscribe(context.Background(), sub.ID)
								test.That(t, err, test.ShouldBeNil)
							}()

							select {
							case <-timeoutCtx.Done():
								t.Log("timed out waiting for packets")
								t.FailNow()
							case <-cancelCtx.Done():
								// We got packets and are happy
							}
						} else {
							_, err := vcs.SubscribeRTP(timeoutCtx, 512, func(_ []*rtp.Packet) {
								t.Log("should not happen")
								t.FailNow()
							})
							test.That(t, err, test.ShouldBeError, tc.expectSubscribeError)
						}
					})
				}
			})
		})
	})
}

func TestRTSPConfig(t *testing.T) {
	// success
	rtspConf := &Config{Address: "rtsp://example.com:5000"}
	_, _, err := rtspConf.Validate("path")
	test.That(t, err, test.ShouldBeNil)
	// badly formatted rtsp address
	rtspConf = &Config{Address: "http://example.com"}
	_, _, err = rtspConf.Validate("path")
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "unsupported scheme")
	rtspConf = &Config{
		Address: "rtsp://example.com:5000",
	}
	_, _, err = rtspConf.Validate("path")
	test.That(t, err, test.ShouldBeNil)
	// test valid transports list
	rtspConf = &Config{
		Address:    "rtsp://example.com:5000",
		Transports: []string{"tcp", "udp"},
	}
	_, _, err = rtspConf.Validate("path")
	test.That(t, err, test.ShouldBeNil)
	// test invalid transports list
	rtspConf = &Config{
		Address:    "rtsp://example.com:5000",
		Transports: []string{"tcp", "invalid"},
	}
	_, _, err = rtspConf.Validate("path")
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "invalid transport")
}

// Dedicated test for performance benchmarking.
func TestRTSPCameraPerformance(t *testing.T) {
	SetLibAVLogLevelFatal()
	logger := logging.NewTestLogger(t)
	bURL, err := base.ParseURL("rtsp://127.0.0.1:32512")
	test.That(t, err, test.ShouldBeNil)

	t.Run("PerformanceTestGetImage", func(t *testing.T) {
		forma := &format.H264{
			PayloadTyp:        96,
			PacketizationMode: 1,
			SPS: []uint8{
				0x67, 0x64, 0x00, 0x15, 0xac, 0xb2, 0x03, 0xc1,
				0x1f, 0xd6, 0x02, 0xdc, 0x08, 0x08, 0x16, 0x94,
				0x00, 0x00, 0x03, 0x00, 0x04, 0x00, 0x00, 0x03,
				0x00, 0xf0, 0x3c, 0x58, 0xb9, 0x20,
			},
			PPS: []uint8{0x68, 0xeb, 0xc3, 0xcb, 0x22, 0xc0},
		}

		h, closeFunc := NewMockH264ServerHandler(t, forma, bURL, logger)
		defer closeFunc()
		test.That(t, h.S.Start(), test.ShouldBeNil)

		timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), time.Second*10)
		defer timeoutCancel()
		config := resource.NewEmptyConfig(camera.Named("foo"), ModelAgnostic)
		config.ConvertedAttributes = &Config{Address: "rtsp://" + h.S.RTSPAddress + "/stream1"}
		rtspCam, err := NewRTSPCamera(timeoutCtx, nil, config, logger)
		test.That(t, err, test.ShouldBeNil)
		defer func() { test.That(t, rtspCam.Close(context.Background()), test.ShouldBeNil) }()

		imageTimeoutCtx, imageTimeoutCancel := context.WithTimeout(context.Background(), time.Second*10)
		defer imageTimeoutCancel()

		const iterations = 100
		var im image.Image
		var frameAvailable bool

		// A loop to keep trying to get the first image until a frame is available.
		for {
			img, err := camera.DecodeImageFromCamera(imageTimeoutCtx, rtspCam, nil, nil)
			if err == nil && img != nil {
				im = img
				frameAvailable = true
				break
			}

			if imageTimeoutCtx.Err() != nil {
				t.Fatalf("Timeout waiting for the first frame")
			}
		}

		// Validate the first retrieved image
		test.That(t, im.Bounds(), test.ShouldResemble, image.Rect(0, 0, 480, 270))

		if !frameAvailable {
			t.Fatal("No frame became available before starting the performance test")
		}

		// Performance testing: Loop over multiple GetImage calls
		for range make([]int, iterations) {
			start := time.Now()
			namedImages, metadata, err := rtspCam.Images(timeoutCtx, nil, nil)
			test.That(t, err, test.ShouldBeNil)
			test.That(t, len(namedImages), test.ShouldEqual, 1)

			bytes, err := namedImages[0].Bytes(timeoutCtx)
			test.That(t, err, test.ShouldBeNil)
			test.That(t, len(bytes), test.ShouldBeGreaterThan, 0)
			test.That(t, namedImages[0].MimeType(), test.ShouldEqual, rutils.MimeTypeJPEG)
			test.That(t, metadata.CapturedAt, test.ShouldHappenBefore, time.Now())
			test.That(t, metadata.CapturedAt, test.ShouldHappenAfter, start)

			time.Sleep(50 * time.Millisecond)
		}
	})
}

func TestUPNPStuff(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger(t)

	ctx = context.WithValue(ctx,
		viamupnp.FindAllTestKey,
		[]viamupnp.UPNPDevice{
			{Service: ssdp.Service{Location: "http://eliot:12312/asd.xml"}, Desc: nil},
		},
	)

	u, err := base.ParseURL("rtsp://a:b@UPNP_DISCOVER/abc")
	test.That(t, err, test.ShouldBeNil)
	test.That(t, u.Host, test.ShouldEqual, "UPNP_DISCOVER")

	c := Config{
		Address: "rtsp://a:b@foo/abc",
	}
	u, err = c.parseAndFixAddress(ctx, logger)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, u.Host, test.ShouldEqual, "foo")

	c.Address = "rtsp://a:b@UPNP_DISCOVER/abc"
	u, err = c.parseAndFixAddress(ctx, logger)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, u.Host, test.ShouldEqual, "eliot")

	c.Address = "rtsp://a:b@UPNP_DISCOVER:1234/abc"
	u, err = c.parseAndFixAddress(ctx, logger)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, u.Host, test.ShouldEqual, "eliot:1234")
}
