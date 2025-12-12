// Package unifi provides Unifi discovery for RTSP cameras
package unifi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/discovery"
	"go.viam.com/test"
)

func TestDiscoveryConfig(t *testing.T) {
	t.Run("Test Empty Config", func(t *testing.T) {
		cfg := Config{}
		deps, _, err := cfg.Validate("")
		test.That(t, err, test.ShouldNotBeNil)
		test.That(t, err.Error(), test.ShouldContainSubstring, "nvr_address is required")
		test.That(t, deps, test.ShouldBeNil)
	})

	t.Run("Test Missing Token", func(t *testing.T) {
		cfg := Config{NVRAddress: "10.0.0.1"}
		deps, _, err := cfg.Validate("")
		test.That(t, err, test.ShouldNotBeNil)
		test.That(t, err.Error(), test.ShouldContainSubstring, "unifi_token is required")
		test.That(t, deps, test.ShouldBeNil)
	})

	t.Run("Test Valid Config", func(t *testing.T) {
		cfg := Config{NVRAddress: "10.0.0.1", UnifiToken: "test-token"}
		deps, _, err := cfg.Validate("")
		test.That(t, err, test.ShouldBeNil)
		test.That(t, deps, test.ShouldBeNil)
	})
}

func TestDiscoveryService(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger(t)

	t.Run("Test Service Creation", func(t *testing.T) {
		testName := "test-unifi"
		cfg := Config{NVRAddress: "10.0.0.1", UnifiToken: "test-token"}
		resourceCfg := resource.Config{
			API:                 discovery.API,
			Model:               Model,
			Name:                testName,
			ConvertedAttributes: &cfg,
		}
		dis, err := newUnifiDiscovery(ctx, nil, resourceCfg, logger)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, dis, test.ShouldNotBeNil)
		test.That(t, dis.Name().ShortName(), test.ShouldEqual, testName)
	})
}

func TestConvertRTSPStoRTSP(t *testing.T) {
	t.Run("Test standard conversion", func(t *testing.T) {
		rtspsURL := "rtsps://10.1.14.106:7441/6uVHv61ad7NDfMCS?enableSrtp"
		expected := "rtsp://10.1.14.106:7447/6uVHv61ad7NDfMCS"
		result := convertRTSPStoRTSP(rtspsURL)
		test.That(t, result, test.ShouldEqual, expected)
	})

	t.Run("Test without enableSrtp query param", func(t *testing.T) {
		rtspsURL := "rtsps://10.1.14.106:7441/abcdef123456"
		expected := "rtsp://10.1.14.106:7447/abcdef123456"
		result := convertRTSPStoRTSP(rtspsURL)
		test.That(t, result, test.ShouldEqual, expected)
	})

	t.Run("Test already rtsp URL", func(t *testing.T) {
		// If given an rtsp URL, it should still work (just port change)
		rtspURL := "rtsp://10.1.14.106:7441/stream"
		expected := "rtsp://10.1.14.106:7447/stream"
		result := convertRTSPStoRTSP(rtspURL)
		test.That(t, result, test.ShouldEqual, expected)
	})
}

func TestSanitizeName(t *testing.T) {
	t.Run("Test lowercase and spaces with ID", func(t *testing.T) {
		name := "G5 Turret Ultra"
		id := "692f41440040a303e405534c"
		expected := "g5_turret_ultra_692f41"
		result := sanitizeName(name, id)
		test.That(t, result, test.ShouldEqual, expected)
	})

	t.Run("Test already lowercase with ID", func(t *testing.T) {
		name := "camera_1"
		id := "abc123def456"
		expected := "camera_1_abc123"
		result := sanitizeName(name, id)
		test.That(t, result, test.ShouldEqual, expected)
	})

	t.Run("Test multiple spaces with ID", func(t *testing.T) {
		name := "Front Door Camera"
		id := "cam1234567"
		expected := "front_door_camera_cam123"
		result := sanitizeName(name, id)
		test.That(t, result, test.ShouldEqual, expected)
	})

	t.Run("Test short ID", func(t *testing.T) {
		name := "Camera"
		id := "abc"
		expected := "camera_abc"
		result := sanitizeName(name, id)
		test.That(t, result, test.ShouldEqual, expected)
	})

	t.Run("Test empty ID", func(t *testing.T) {
		name := "Camera"
		id := ""
		expected := "camera"
		result := sanitizeName(name, id)
		test.That(t, result, test.ShouldEqual, expected)
	})
}

func TestDiscoverResources(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger(t)

	t.Run("Test successful discovery", func(t *testing.T) {
		// Create mock server for cameras endpoint
		camerasResponse := []unifiCamera{
			{ID: "cam1", Name: "Front Door", State: "CONNECTED"},
			{ID: "cam2", Name: "Backyard", State: "CONNECTED"},
		}

		streamResponses := map[string]rtspStreamResponse{
			"cam1": {High: "rtsps://10.0.0.1:7441/stream1?enableSrtp"},
			"cam2": {High: "rtsps://10.0.0.1:7441/stream2?enableSrtp"},
		}

		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check API key header
			if r.Header.Get("X-Api-Key") != "test-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			switch r.URL.Path {
			case "/proxy/protect/integration/v1/cameras":
				w.Header().Set("Content-Type", "application/json")
				err := json.NewEncoder(w).Encode(camerasResponse)
				test.That(t, err, test.ShouldBeNil)
			case "/proxy/protect/integration/v1/cameras/cam1/rtsps-stream":
				w.Header().Set("Content-Type", "application/json")
				err := json.NewEncoder(w).Encode(streamResponses["cam1"])
				test.That(t, err, test.ShouldBeNil)
			case "/proxy/protect/integration/v1/cameras/cam2/rtsps-stream":
				w.Header().Set("Content-Type", "application/json")
				err := json.NewEncoder(w).Encode(streamResponses["cam2"])
				test.That(t, err, test.ShouldBeNil)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		// Extract host from server URL (remove https://)
		host := server.URL[8:] // Remove "https://"

		dis := &unifiDiscovery{
			Named:     resource.NewName(discovery.API, "test").AsNamed(),
			unifToken: "test-token",
			nvrAddr:   host,
			logger:    logger,
		}
		// Override httpClient to use test server's client
		dis.httpClient = server.Client()

		configs, err := dis.DiscoverResources(ctx, nil)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, len(configs), test.ShouldEqual, 2)

		// Check first camera (ID: cam1 -> last 6 chars is "1" but ID is short, so full ID used)
		test.That(t, configs[0].Name, test.ShouldEqual, "front_door_cam1")
		rtspAddr, ok := configs[0].Attributes["rtsp_address"]
		test.That(t, ok, test.ShouldBeTrue)
		test.That(t, rtspAddr, test.ShouldContainSubstring, "rtsp://")
		test.That(t, rtspAddr, test.ShouldContainSubstring, ":7447/")

		// Check second camera (ID: cam2)
		test.That(t, configs[1].Name, test.ShouldEqual, "backyard_cam2")
	})

	t.Run("Test camera with no RTSP stream", func(t *testing.T) {
		camerasResponse := []unifiCamera{
			{ID: "cam1", Name: "No Stream Camera", State: "CONNECTED"},
		}

		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/proxy/protect/integration/v1/cameras":
				w.Header().Set("Content-Type", "application/json")
				err := json.NewEncoder(w).Encode(camerasResponse)
				test.That(t, err, test.ShouldBeNil)
			case "/proxy/protect/integration/v1/cameras/cam1/rtsps-stream":
				// Return empty stream response
				w.Header().Set("Content-Type", "application/json")
				err := json.NewEncoder(w).Encode(rtspStreamResponse{})
				test.That(t, err, test.ShouldBeNil)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		host := server.URL[8:]

		dis := &unifiDiscovery{
			Named:     resource.NewName(discovery.API, "test").AsNamed(),
			unifToken: "test-token",
			nvrAddr:   host,
			logger:    logger,
		}
		dis.httpClient = server.Client()

		configs, err := dis.DiscoverResources(ctx, nil)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, len(configs), test.ShouldEqual, 0) // No cameras with valid streams
	})

	t.Run("Test API error", func(t *testing.T) {
		testAPIError(ctx, t, logger, http.StatusInternalServerError, "test-token", "API returned status 500")
	})

	t.Run("Test unauthorized error", func(t *testing.T) {
		testAPIError(ctx, t, logger, http.StatusUnauthorized, "bad-token", "API returned status 401")
	})
}

func testAPIError(ctx context.Context, t *testing.T, logger logging.Logger, statusCode int, token, expectedErr string) {
	t.Helper()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(statusCode)
	}))
	defer server.Close()

	host := server.URL[8:]

	dis := &unifiDiscovery{
		Named:     resource.NewName(discovery.API, "test").AsNamed(),
		unifToken: token,
		nvrAddr:   host,
		logger:    logger,
	}
	dis.httpClient = server.Client()

	configs, err := dis.DiscoverResources(ctx, nil)
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, expectedErr)
	test.That(t, configs, test.ShouldBeNil)
}

func TestGetCameras(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger(t)

	t.Run("Test successful get cameras", func(t *testing.T) {
		camerasResponse := []unifiCamera{
			{ID: "cam1", Name: "Camera 1", State: "CONNECTED"},
			{ID: "cam2", Name: "Camera 2", State: "DISCONNECTED"},
		}

		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			test.That(t, r.URL.Path, test.ShouldEqual, "/proxy/protect/integration/v1/cameras")
			test.That(t, r.Header.Get("X-Api-Key"), test.ShouldEqual, "test-token")
			test.That(t, r.Header.Get("Accept"), test.ShouldEqual, "application/json")

			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(camerasResponse)
			test.That(t, err, test.ShouldBeNil)
		}))
		defer server.Close()

		host := server.URL[8:]

		dis := &unifiDiscovery{
			Named:     resource.NewName(discovery.API, "test").AsNamed(),
			unifToken: "test-token",
			nvrAddr:   host,
			logger:    logger,
		}
		dis.httpClient = server.Client()

		cameras, err := dis.getCameras(ctx)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, len(cameras), test.ShouldEqual, 2)
		test.That(t, cameras[0].ID, test.ShouldEqual, "cam1")
		test.That(t, cameras[0].Name, test.ShouldEqual, "Camera 1")
		test.That(t, cameras[1].ID, test.ShouldEqual, "cam2")
	})

	t.Run("Test empty cameras list", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode([]unifiCamera{})
			test.That(t, err, test.ShouldBeNil)
		}))
		defer server.Close()

		host := server.URL[8:]

		dis := &unifiDiscovery{
			Named:     resource.NewName(discovery.API, "test").AsNamed(),
			unifToken: "test-token",
			nvrAddr:   host,
			logger:    logger,
		}
		dis.httpClient = server.Client()

		cameras, err := dis.getCameras(ctx)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, len(cameras), test.ShouldEqual, 0)
	})
}

func TestGetRTSPStream(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger(t)

	t.Run("Test get high quality stream", func(t *testing.T) {
		streamResponse := rtspStreamResponse{
			High:   "rtsps://10.0.0.1:7441/highstream?enableSrtp",
			Medium: "rtsps://10.0.0.1:7441/medstream?enableSrtp",
			Low:    "rtsps://10.0.0.1:7441/lowstream?enableSrtp",
		}

		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			test.That(t, r.URL.Path, test.ShouldEqual, "/proxy/protect/integration/v1/cameras/cam123/rtsps-stream")
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(streamResponse)
			test.That(t, err, test.ShouldBeNil)
		}))
		defer server.Close()

		host := server.URL[8:]

		dis := &unifiDiscovery{
			Named:     resource.NewName(discovery.API, "test").AsNamed(),
			unifToken: "test-token",
			nvrAddr:   host,
			logger:    logger,
		}
		dis.httpClient = server.Client()

		rtspURL, err := dis.getRTSPStream(ctx, "cam123")
		test.That(t, err, test.ShouldBeNil)
		test.That(t, rtspURL, test.ShouldEqual, "rtsp://10.0.0.1:7447/highstream")
	})

	t.Run("Test fallback to medium stream", func(t *testing.T) {
		testStreamFallback(ctx, t, logger, rtspStreamResponse{
			High:   "",
			Medium: "rtsps://10.0.0.1:7441/medstream?enableSrtp",
			Low:    "rtsps://10.0.0.1:7441/lowstream?enableSrtp",
		}, "rtsp://10.0.0.1:7447/medstream")
	})

	t.Run("Test fallback to low stream", func(t *testing.T) {
		testStreamFallback(ctx, t, logger, rtspStreamResponse{
			High:   "",
			Medium: "",
			Low:    "rtsps://10.0.0.1:7441/lowstream?enableSrtp",
		}, "rtsp://10.0.0.1:7447/lowstream")
	})

	t.Run("Test no streams available", func(t *testing.T) {
		testStreamFallback(ctx, t, logger, rtspStreamResponse{}, "")
	})
}

func testStreamFallback(ctx context.Context, t *testing.T, logger logging.Logger, resp rtspStreamResponse, expected string) {
	t.Helper()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(resp)
		test.That(t, err, test.ShouldBeNil)
	}))
	defer server.Close()

	host := server.URL[8:]

	dis := &unifiDiscovery{
		Named:     resource.NewName(discovery.API, "test").AsNamed(),
		unifToken: "test-token",
		nvrAddr:   host,
		logger:    logger,
	}
	dis.httpClient = server.Client()

	rtspURL, err := dis.getRTSPStream(ctx, "cam123")
	test.That(t, err, test.ShouldBeNil)
	test.That(t, rtspURL, test.ShouldEqual, expected)
}

func TestCheckResponse(t *testing.T) {
	t.Run("Test 200 OK with JSON", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}
		err := checkResponse(resp)
		test.That(t, err, test.ShouldBeNil)
	})

	t.Run("Test 200 OK with JSON charset", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json; charset=utf-8"}},
		}
		err := checkResponse(resp)
		test.That(t, err, test.ShouldBeNil)
	})

	t.Run("Test 401 Unauthorized", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusUnauthorized,
			Header:     http.Header{},
		}
		err := checkResponse(resp)
		test.That(t, err, test.ShouldNotBeNil)
		test.That(t, err.Error(), test.ShouldContainSubstring, "authentication failed")
	})

	t.Run("Test 500 Internal Server Error", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusInternalServerError,
			Header:     http.Header{},
		}
		err := checkResponse(resp)
		test.That(t, err, test.ShouldNotBeNil)
		test.That(t, err.Error(), test.ShouldContainSubstring, "500")
	})

	t.Run("Test 200 with HTML content type", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/html"}},
		}
		err := checkResponse(resp)
		test.That(t, err, test.ShouldNotBeNil)
		test.That(t, err.Error(), test.ShouldContainSubstring, "unexpected content type")
	})

	t.Run("Test 200 with empty content type", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
		}
		err := checkResponse(resp)
		test.That(t, err, test.ShouldNotBeNil)
		test.That(t, err.Error(), test.ShouldContainSubstring, "unexpected content type")
	})
}
