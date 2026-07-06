package garmin

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/viam-modules/viamrtsp"
	"github.com/viamrobotics/zeroconf"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/test"
)

// newServiceEntry builds a zeroconf.ServiceEntry as the resolver would yield one.
func newServiceEntry(instance, hostname, ipv4 string) *zeroconf.ServiceEntry {
	entry := zeroconf.NewServiceEntry(instance, mdnsService, mdnsDomain)
	entry.HostName = hostname
	if ipv4 != "" {
		entry.AddrIPv4 = []net.IP{net.ParseIP(ipv4)}
	}
	return entry
}

func TestConfigDefaults(t *testing.T) {
	t.Run("empty config uses garmin defaults", func(t *testing.T) {
		cfg := &Config{}
		test.That(t, cfg.port(), test.ShouldEqual, defaultRTSPPort)
		test.That(t, cfg.paths(), test.ShouldResemble, defaultStreamPaths)
	})

	t.Run("overrides are respected", func(t *testing.T) {
		cfg := &Config{RTSPPort: 9000, StreamPaths: []string{"/a", "/b"}}
		test.That(t, cfg.port(), test.ShouldEqual, 9000)
		test.That(t, cfg.paths(), test.ShouldResemble, []string{"/a", "/b"})
	})
}

func TestConfigValidate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		_, _, err := (&Config{RTSPPort: 8554, StreamPaths: []string{"/x"}}).Validate("")
		test.That(t, err, test.ShouldBeNil)
	})
	t.Run("port out of range", func(t *testing.T) {
		_, _, err := (&Config{RTSPPort: 70000}).Validate("")
		test.That(t, err, test.ShouldNotBeNil)
	})
	t.Run("empty stream path", func(t *testing.T) {
		_, _, err := (&Config{StreamPaths: []string{""}}).Validate("")
		test.That(t, err, test.ShouldNotBeNil)
	})
}

func TestAddressHost(t *testing.T) {
	t.Run("prefers advertised hostname", func(t *testing.T) {
		cam := Camera{Host: "garmin-cv28-x.local", IPs: []net.IP{net.ParseIP("172.16.0.5")}}
		test.That(t, cam.addressHost(), test.ShouldEqual, "garmin-cv28-x.local")
	})
	t.Run("falls back to IP when no hostname", func(t *testing.T) {
		cam := Camera{IPs: []net.IP{net.ParseIP("172.16.0.5")}}
		test.That(t, cam.addressHost(), test.ShouldEqual, "172.16.0.5")
	})
	t.Run("empty when nothing advertised", func(t *testing.T) {
		test.That(t, Camera{}.addressHost(), test.ShouldEqual, "")
	})
}

func TestServiceEntryToCamera(t *testing.T) {
	// HostName comes with a trailing dot from mDNS; it should be stripped.
	cam := serviceEntryToCamera(newServiceEntry("garmin-cv28-copepod-3530195020", "garmin-cv28-copepod-3530195020.local.", "172.16.0.5"))
	test.That(t, cam.Instance, test.ShouldEqual, "garmin-cv28-copepod-3530195020")
	test.That(t, cam.Host, test.ShouldEqual, "garmin-cv28-copepod-3530195020.local")
	test.That(t, cam.IPs[0].String(), test.ShouldEqual, "172.16.0.5")
}

func TestCollectCamerasReturnsOnClosedChannel(t *testing.T) {
	logger := logging.NewTestLogger(t)

	entries := make(chan *zeroconf.ServiceEntry, 4)
	entries <- newServiceEntry("garmin-a", "garmin-a.local.", "172.16.0.1")
	entries <- newServiceEntry("garmin-b", "garmin-b.local.", "172.16.0.2")
	close(entries)

	// ctx never expires: before the J1 fix, a single-value receive on the closed
	// channel yields nil forever and spins the loop at 100% CPU. The comma-ok
	// receive must instead finalize and return. Guard with a timeout so a
	// regression fails fast rather than hanging the whole suite.
	done := make(chan []Camera, 1)
	go func() {
		done <- collectCameras(context.Background(), entries, logger)
	}()

	select {
	case cams := <-done:
		test.That(t, len(cams), test.ShouldEqual, 2)
	case <-time.After(2 * time.Second):
		t.Fatal("collectCameras did not return on closed channel (busy-loop regression)")
	}
}

func TestCollectCamerasDrainsBufferedEntriesAtTimeout(t *testing.T) {
	logger := logging.NewTestLogger(t)

	// Pre-buffer several distinct entries, then use an already-expired ctx so the
	// Done case is always ready. Without the drain, the outer select would pick
	// Done over a ready receive and drop buffered cameras (with n=16, the odds of
	// draining all before the coin-flip lands on Done are ~1/2^16). The drain
	// must collect every buffered camera.
	const n = 16
	entries := make(chan *zeroconf.ServiceEntry, n)
	for i := 0; i < n; i++ {
		instance := fmt.Sprintf("garmin-%d", i)
		entries <- newServiceEntry(instance, instance+".local.", "172.16.0.1")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cams := collectCameras(ctx, entries, logger)
	test.That(t, len(cams), test.ShouldEqual, n)
}

func TestCameraName(t *testing.T) {
	t.Run("single path uses sanitized instance", func(t *testing.T) {
		test.That(t, cameraName("garmin-cv28-copepod-3530195020", "/Independent/1080p", false),
			test.ShouldEqual, "garmin-cv28-copepod-3530195020")
	})
	t.Run("multi path appends sanitized path", func(t *testing.T) {
		test.That(t, cameraName("garmin-cv28", "/Independent/1080p", true),
			test.ShouldEqual, "garmin-cv28-Independent-1080p")
	})
	t.Run("strips invalid characters", func(t *testing.T) {
		test.That(t, cameraName("garmin cv28!@#copepod", "/x", false),
			test.ShouldEqual, "garmin-cv28-copepod")
	})
}

func TestCamerasToConfigs(t *testing.T) {
	logger := logging.NewTestLogger(t)

	t.Run("default config emits one camera config per route", func(t *testing.T) {
		cams := []Camera{{Instance: "garmin-cv28-x", Host: "garmin-cv28-x.local"}}
		configs, err := camerasToConfigs(cams, &Config{}, logger)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, len(configs), test.ShouldEqual, len(defaultStreamPaths))

		// Each config points at a distinct route and has a unique name.
		gotAddrs := map[string]bool{}
		gotNames := map[string]bool{}
		for _, cfg := range configs {
			test.That(t, cfg.API, test.ShouldResemble, camera.API)
			test.That(t, cfg.Model, test.ShouldResemble, viamrtsp.ModelAgnostic)
			rtsp, ok := cfg.ConvertedAttributes.(*viamrtsp.Config)
			test.That(t, ok, test.ShouldBeTrue)
			test.That(t, *rtsp.RTPPassthrough, test.ShouldBeTrue)
			gotAddrs[rtsp.Address] = true
			gotNames[cfg.Name] = true
		}
		test.That(t, len(gotAddrs), test.ShouldEqual, len(defaultStreamPaths))
		test.That(t, len(gotNames), test.ShouldEqual, len(defaultStreamPaths))
		test.That(t, gotAddrs["rtsp://garmin-cv28-x.local:8554/Independent/1080p"], test.ShouldBeTrue)
	})

	t.Run("multiple paths emit a config per path with unique names", func(t *testing.T) {
		cams := []Camera{{Instance: "garmin-cv28-x", Host: "garmin-cv28-x.local"}}
		cfg := &Config{StreamPaths: []string{"/Independent/1080p", "/Independent/720p"}}
		configs, err := camerasToConfigs(cams, cfg, logger)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, len(configs), test.ShouldEqual, 2)
		test.That(t, configs[0].Name, test.ShouldNotEqual, configs[1].Name)
	})

	t.Run("falls back to IP when no hostname advertised", func(t *testing.T) {
		cams := []Camera{{Instance: "garmin-cv28-x", IPs: []net.IP{net.ParseIP("172.16.0.5")}}}
		configs, err := camerasToConfigs(cams, &Config{StreamPaths: []string{"/Independent/720p"}}, logger)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, len(configs), test.ShouldEqual, 1)
		rtsp := configs[0].ConvertedAttributes.(*viamrtsp.Config)
		test.That(t, rtsp.Address, test.ShouldEqual, "rtsp://172.16.0.5:8554/Independent/720p")
	})

	t.Run("skips cameras with neither hostname nor IP", func(t *testing.T) {
		cams := []Camera{{Instance: "garmin-cv28-x"}}
		configs, err := camerasToConfigs(cams, &Config{}, logger)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, configs, test.ShouldBeEmpty)
	})
}

func TestCameraConfigMarshalsToMachineConfig(t *testing.T) {
	// The debug binary dumps these configs as JSON; lock the shape a user sees.
	logger := logging.NewTestLogger(t)
	configs, err := camerasToConfigs(
		[]Camera{{Instance: "garmin-cv28-copepod-3530195020", Host: "garmin-cv28-copepod-3530195020.local"}},
		&Config{StreamPaths: []string{"/Independent/720p"}}, logger,
	)
	test.That(t, err, test.ShouldBeNil)

	jsonBytes, err := json.Marshal(configs[0])
	test.That(t, err, test.ShouldBeNil)

	var got map[string]any
	test.That(t, json.Unmarshal(jsonBytes, &got), test.ShouldBeNil)
	test.That(t, got["name"], test.ShouldEqual, "garmin-cv28-copepod-3530195020")
	test.That(t, got["api"], test.ShouldEqual, "rdk:component:camera")
	test.That(t, got["model"], test.ShouldEqual, "viam:viamrtsp:rtsp")
	attrs, ok := got["attributes"].(map[string]any)
	test.That(t, ok, test.ShouldBeTrue)
	test.That(t, attrs["rtsp_address"], test.ShouldEqual, "rtsp://garmin-cv28-copepod-3530195020.local:8554/Independent/720p")
	test.That(t, attrs["rtp_passthrough"], test.ShouldBeTrue)
}

func TestCreateCameraConfig(t *testing.T) {
	cfg, err := createCameraConfig("cam0", "rtsp://host:8554/path")
	test.That(t, err, test.ShouldBeNil)
	test.That(t, cfg.Name, test.ShouldEqual, "cam0")
	// Attributes map must round-trip the address so viam-server can read it raw.
	test.That(t, cfg.Attributes["rtsp_address"], test.ShouldEqual, "rtsp://host:8554/path")

	var _ resource.Config = cfg
}
