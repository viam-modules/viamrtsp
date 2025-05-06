// Package viamonvif provides the discovery service that wraps ONVIF integration for the viamrtsp module
package viamonvif

import (
	"bytes"
	"context"
	"encoding/base64"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/viam-modules/viamrtsp"
	"github.com/viam-modules/viamrtsp/viamonvif/device"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/discovery"
	"go.viam.com/test"
)

func TestDiscoveryService(t *testing.T) {
	cfg := Config{}
	ctx := context.Background()
	logger := logging.NewTestLogger(t)

	t.Run("Test Default Service with no cameras", func(t *testing.T) {
		testName := "test"
		resourceCfg := resource.Config{API: discovery.API, Model: Model, Name: testName, ConvertedAttributes: &cfg}
		dis, err := newDiscovery(ctx, nil, resourceCfg, logger)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, dis, test.ShouldNotBeNil)
		test.That(t, dis.Name().ShortName(), test.ShouldResemble, testName)
		cfgs, err := dis.DiscoverResources(ctx, nil)
		test.That(t, cfgs, test.ShouldBeEmpty)
		test.That(t, err, test.ShouldBeError, errNoCamerasFound)
	})
}

func TestCamConfig(t *testing.T) {
	camName := "my-cam"
	camURL := "my-cam-url"
	conf, err := createCameraConfig(camName, camURL, "")
	test.That(t, err, test.ShouldBeNil)
	test.That(t, conf.Name, test.ShouldEqual, camName)
	cfg, err := resource.NativeConfig[*viamrtsp.Config](conf)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, cfg.Address, test.ShouldEqual, camURL)
	test.That(t, *cfg.RTPPassthrough, test.ShouldBeTrue)
	test.That(t, cfg.DiscoveryDep, test.ShouldEqual, "")

	discSvcDep := "discovery-service-dependency"
	conf, err = createCameraConfig(camName, camURL, discSvcDep)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, conf.Name, test.ShouldEqual, camName)
	cfg, err = resource.NativeConfig[*viamrtsp.Config](conf)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, cfg.Address, test.ShouldEqual, camURL)
	test.That(t, *cfg.RTPPassthrough, test.ShouldBeTrue)
	test.That(t, cfg.DiscoveryDep, test.ShouldEqual, discSvcDep)
}

func TestDiscoveryConfig(t *testing.T) {
	t.Run("Test Empty Config", func(t *testing.T) {
		cfg := Config{}
		deps, err := cfg.Validate("")
		test.That(t, err, test.ShouldBeNil)
		test.That(t, deps, test.ShouldBeEmpty)
	})
	t.Run("Test Valid config", func(t *testing.T) {
		cfg := Config{Credentials: []device.Credentials{
			{User: "user1", Pass: "pass1"},
			{User: "user2", Pass: "pass2"},
			{User: "user3", Pass: ""},
			{User: "", Pass: ""},
		}}
		deps, err := cfg.Validate("")
		test.That(t, err, test.ShouldBeNil)
		test.That(t, deps, test.ShouldBeEmpty)
	})
	t.Run("Test Invalid Config", func(t *testing.T) {
		cfg := Config{Credentials: []device.Credentials{{User: "", Pass: "pass1"}}}
		deps, err := cfg.Validate("")
		test.That(t, err.Error(), test.ShouldContainSubstring, "credential missing username, has password pass1")
		test.That(t, deps, test.ShouldBeEmpty)
	})
}

func TestGetCredFromExtra(t *testing.T) {
	t.Run("Test good extra with User and Pass as strings", func(t *testing.T) {
		extra := map[string]any{
			"User": "user",
			"Pass": "pass",
		}
		cred, ok := getCredFromExtra(extra)
		test.That(t, cred.User, test.ShouldEqual, "user")
		test.That(t, cred.Pass, test.ShouldEqual, "pass")
		test.That(t, ok, test.ShouldBeTrue)
	})
	t.Run("Test good extra with no Pass", func(t *testing.T) {
		extra := map[string]any{
			"User": "user",
		}
		cred, ok := getCredFromExtra(extra)
		test.That(t, cred.User, test.ShouldEqual, "user")
		test.That(t, cred.Pass, test.ShouldEqual, "")
		test.That(t, ok, test.ShouldBeTrue)
	})
	t.Run("Test bad extra with no strings", func(t *testing.T) {
		extra := map[string]any{
			"User": 1,
			"Pass": true,
		}
		cred, ok := getCredFromExtra(extra)
		test.That(t, cred.User, test.ShouldEqual, "")
		test.That(t, cred.Pass, test.ShouldEqual, "")
		test.That(t, ok, test.ShouldBeFalse)
	})
	t.Run("Test nil cred", func(t *testing.T) {
		cred, ok := getCredFromExtra(nil)
		test.That(t, cred.User, test.ShouldEqual, "")
		test.That(t, cred.Pass, test.ShouldEqual, "")
		test.That(t, ok, test.ShouldBeFalse)
	})
}

func TestDoCommandPreview(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger(t)

	t.Run("Test preview command with valid RTSP URL", func(t *testing.T) {
		server := startTestHTTPServer(t, "/snapshot", http.StatusOK, "image/jpeg", "mockImageData", false)
		defer server.Close()

		dis := &rtspDiscovery{
			rtspToSnapshotURIs: map[string]string{
				"rtsp://camera1/stream": server.URL + "/snapshot",
			},
			logger: logger,
		}

		command := map[string]interface{}{
			"command": "preview",
			"attributes": map[string]interface{}{
				"rtsp_address": "rtsp://camera1/stream",
			},
		}

		result, err := dis.DoCommand(ctx, command)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, result["preview"], test.ShouldEqual, "data:image/jpeg;base64,bW9ja0ltYWdlRGF0YQ==")
	})

	t.Run("Test preview command with valid RTSP URL and https stream uri", func(t *testing.T) {
		server := startTestHTTPServer(t, "/snapshot", http.StatusOK, "image/jpeg", "mockImageData", true)
		defer server.Close()

		dis := &rtspDiscovery{
			rtspToSnapshotURIs: map[string]string{
				"rtsp://camera1/stream": server.URL + "/snapshot",
			},
			logger: logger,
		}

		command := map[string]interface{}{
			"command": "preview",
			"attributes": map[string]interface{}{
				"rtsp_address": "rtsp://camera1/stream",
			},
		}
		result, err := dis.DoCommand(ctx, command)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, result["preview"], test.ShouldEqual, "data:image/jpeg;base64,bW9ja0ltYWdlRGF0YQ==")
	})

	t.Run("Test preview command when streaming to snapshot mapping does not exist", func(t *testing.T) {
		dis := &rtspDiscovery{
			rtspToSnapshotURIs: map[string]string{
				"rtsp://camera1/stream": "http://invalid/snapshot",
			},
			logger: logger,
		}

		command := map[string]interface{}{
			"command": "preview",
			"attributes": map[string]interface{}{
				"rtsp_address": "rtsp://invalid/stream",
			},
		}

		result, err := dis.DoCommand(ctx, command)
		test.That(t, err, test.ShouldNotBeNil)
		test.That(t, err.Error(), test.ShouldContainSubstring, "snapshot URI not found")
		test.That(t, result, test.ShouldBeNil)
	})

	t.Run("Test preview command with snapshot download HTTP error", func(t *testing.T) {
		server := startTestHTTPServer(t, "/snapshot", http.StatusInternalServerError, "text/plain", "Internal Server Error", false)
		defer server.Close()

		dis := &rtspDiscovery{
			rtspToSnapshotURIs: map[string]string{
				"rtsp://camera1/stream": server.URL + "/snapshot",
			},
			logger: logger,
		}

		command := map[string]interface{}{
			"command": "preview",
			"attributes": map[string]interface{}{
				"rtsp_address": "rtsp://camera1/stream",
			},
		}

		result, err := dis.DoCommand(ctx, command)
		test.That(t, err, test.ShouldNotBeNil)
		test.That(t, err.Error(), test.ShouldContainSubstring, "500: Internal Server Error")
		test.That(t, result, test.ShouldBeNil)
	})

	t.Run("Test successful preview command with broken snapshot URI and valid streaming URI", func(t *testing.T) {
		logger := logging.NewTestLogger(t)

		bURL, err := base.ParseURL("rtsp://127.0.0.1:32512")
		test.That(t, err, test.ShouldBeNil)
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
		h, closeFunc := viamrtsp.NewMockH264ServerHandler(t, forma, bURL, logger)
		defer closeFunc()

		// Start rtsp feed
		test.That(t, h.S.Start(), test.ShouldBeNil)

		rtspAddr := "rtsp://" + h.S.RTSPAddress + "/stream1"
		dis := &rtspDiscovery{
			rtspToSnapshotURIs: map[string]string{
				rtspAddr: "http://invalid/snapshot",
			},
			logger: logger,
		}

		command := map[string]interface{}{
			"command": "preview",
			"attributes": map[string]interface{}{
				"rtsp_address": rtspAddr,
			},
		}
		result, err := dis.DoCommand(ctx, command)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, len(result["preview"].(string)), test.ShouldEqual, 7435)
		imgData, err := base64.StdEncoding.DecodeString(result["preview"].(string)[len("data:image/jpeg;base64,"):])
		test.That(t, err, test.ShouldBeNil)
		jpegImg, err := jpeg.Decode(bytes.NewReader(imgData))
		test.That(t, err, test.ShouldBeNil)
		test.That(t, jpegImg, test.ShouldNotBeNil)
		test.That(t, jpegImg.Bounds().Dx(), test.ShouldEqual, 480)
		test.That(t, jpegImg.Bounds().Dy(), test.ShouldEqual, 270)
	})

	t.Run("Test preview command where both rtsp and snapshot URI fail", func(t *testing.T) {
		dis := &rtspDiscovery{
			rtspToSnapshotURIs: map[string]string{
				"rtsp://invalid/stream": "http://invalid/snapshot",
			},
			logger: logger,
		}

		command := map[string]interface{}{
			"command": "preview",
			"attributes": map[string]interface{}{
				"rtsp_address": "rtsp://invalid/stream",
			},
		}

		result, err := dis.DoCommand(ctx, command)
		test.That(t, err, test.ShouldNotBeNil)
		test.That(t, result, test.ShouldBeNil)
		test.That(t, err.Error(), test.ShouldContainSubstring, "snapshot error")
		test.That(t, err.Error(), test.ShouldContainSubstring, "both snapshot and RTSP fetch failed")
	})
}

func startTestHTTPServer(t *testing.T, path string, statusCode int, contentType, responseBody string, useTLS bool) *httptest.Server {
	handler := http.NewServeMux()
	handler.HandleFunc(path, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(statusCode)
		_, err := w.Write([]byte(responseBody))
		if err != nil {
			t.Logf("failed to write response: %v", err)
		}
	})

	if useTLS {
		return httptest.NewTLSServer(handler)
	}
	return httptest.NewServer(handler)
}
