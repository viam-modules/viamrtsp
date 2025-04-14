// Package viamonvif provides the discovery service that wraps ONVIF integration for the viamrtsp module
package viamonvif

import (
	"context"
	"net"
	"net/http"
	"testing"

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
		server := startTestHTTPServer(t, "/snapshot", http.StatusOK, "image/jpeg", "mockImageData")
		defer server.Close()

		serverURL := "http://" + server.Addr

		dis := &rtspDiscovery{
			URIs: []URI{
				{StreamURI: "rtsp://camera1/stream", SnapshotURI: serverURL + "/snapshot"},
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

	t.Run("Test preview command with invalid RTSP URL", func(t *testing.T) {
		dis := &rtspDiscovery{
			URIs: []URI{
				{StreamURI: "rtsp://camera1/stream", SnapshotURI: "http://invalid/snapshot"},
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

	t.Run("Test preview command with download error", func(t *testing.T) {
		// Start a test HTTP server that returns an error
		server := startTestHTTPServer(t, "/snapshot", http.StatusInternalServerError, "text/plain", "Internal Server Error")
		defer server.Close()

		serverURL := "http://" + server.Addr

		dis := &rtspDiscovery{
			URIs: []URI{
				{StreamURI: "rtsp://camera1/stream", SnapshotURI: serverURL + "/snapshot"},
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
}

func startTestHTTPServer(t *testing.T, path string, statusCode int, contentType, responseBody string) *http.Server {
	handler := http.NewServeMux()
	handler.HandleFunc(path, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(statusCode)
		_, err := w.Write([]byte(responseBody))
		if err != nil {
			t.Logf("failed to write response: %v", err)
		}
	})

	server := &http.Server{Addr: "127.0.0.1:0", Handler: handler}
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		t.Logf("failed to start test HTTP server: %v", err)
	}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			t.Logf("test HTTP server error: %v", err)
		}
	}()

	server.Addr = listener.Addr().String()
	return server
}
