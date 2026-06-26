package garmin

import (
	"net"
	"testing"

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
		test.That(t, cfg.paths(), test.ShouldResemble, []string{defaultStreamPath})
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

	t.Run("default path emits one camera config per camera", func(t *testing.T) {
		cams := []Camera{{Instance: "garmin-cv28-x", Host: "garmin-cv28-x.local"}}
		configs, err := camerasToConfigs(cams, &Config{}, logger)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, len(configs), test.ShouldEqual, 1)

		cfg := configs[0]
		test.That(t, cfg.Name, test.ShouldEqual, "garmin-cv28-x")
		test.That(t, cfg.API, test.ShouldResemble, camera.API)
		test.That(t, cfg.Model, test.ShouldResemble, viamrtsp.ModelAgnostic)

		rtsp, ok := cfg.ConvertedAttributes.(*viamrtsp.Config)
		test.That(t, ok, test.ShouldBeTrue)
		test.That(t, rtsp.Address, test.ShouldEqual, "rtsp://garmin-cv28-x.local:8554/Independent/1080p")
		test.That(t, *rtsp.RTPPassthrough, test.ShouldBeTrue)
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
		configs, err := camerasToConfigs(cams, &Config{}, logger)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, len(configs), test.ShouldEqual, 1)
		rtsp := configs[0].ConvertedAttributes.(*viamrtsp.Config)
		test.That(t, rtsp.Address, test.ShouldEqual, "rtsp://172.16.0.5:8554/Independent/1080p")
	})

	t.Run("skips cameras with neither hostname nor IP", func(t *testing.T) {
		cams := []Camera{{Instance: "garmin-cv28-x"}}
		configs, err := camerasToConfigs(cams, &Config{}, logger)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, configs, test.ShouldBeEmpty)
	})
}

func TestCreateCameraConfig(t *testing.T) {
	cfg, err := createCameraConfig("cam0", "rtsp://host:8554/path")
	test.That(t, err, test.ShouldBeNil)
	test.That(t, cfg.Name, test.ShouldEqual, "cam0")
	// Attributes map must round-trip the address so viam-server can read it raw.
	test.That(t, cfg.Attributes["rtsp_address"], test.ShouldEqual, "rtsp://host:8554/path")

	var _ resource.Config = cfg
}
