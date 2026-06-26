// Package garmin provides a discovery service that finds Garmin marine cameras
// over mDNS and emits ready-to-use viamrtsp camera configs.
//
// Garmin cameras are not ONVIF-discoverable. They advertise themselves over
// mDNS under the "_garmin-mrn-svcm._tcp" service type with a stable *.local
// hostname, and serve an RTSP stream at a fixed path. This service browses for
// those advertisements and turns each camera into a camera resource config,
// removing the manual avahi-browse + hand-built-config workflow.
package garmin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/viam-modules/viamrtsp"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/discovery"
)

// Model is the model for a Garmin discovery service for rtsp cameras.
var Model = viamrtsp.Family.WithModel("garmin")

const (
	// defaultRTSPPort is the port Garmin's GStreamer RTSP server listens on.
	defaultRTSPPort = 8554
	// defaultStreamPath is the RTSP path Garmin cameras serve. Fixed across the
	// models seen in the field; overridable via config if that changes.
	defaultStreamPath = "/Independent/1080p"
)

func init() {
	resource.RegisterService(
		discovery.API,
		Model,
		resource.Registration[discovery.Service, *Config]{
			Constructor: newDiscovery,
		})
}

// Config is the config for the Garmin discovery service.
type Config struct {
	// RTSPPort overrides the RTSP port. Defaults to 8554 if unset.
	RTSPPort int `json:"rtsp_port,omitempty"`
	// StreamPaths overrides the RTSP path(s) to emit per camera. Defaults to
	// ["/Independent/1080p"] if unset. One camera config is emitted per path.
	StreamPaths []string `json:"stream_paths,omitempty"`
}

// Validate validates the discovery service config.
func (cfg *Config) Validate(_ string) ([]string, []string, error) {
	if cfg.RTSPPort < 0 || cfg.RTSPPort > 65535 {
		return nil, nil, fmt.Errorf("rtsp_port %d out of range", cfg.RTSPPort)
	}
	for _, p := range cfg.StreamPaths {
		if p == "" {
			return nil, nil, errors.New("stream_paths entries cannot be empty")
		}
	}
	return []string{}, nil, nil
}

// port returns the configured RTSP port or the default.
func (cfg *Config) port() int {
	if cfg.RTSPPort != 0 {
		return cfg.RTSPPort
	}
	return defaultRTSPPort
}

// paths returns the configured stream paths or the default.
func (cfg *Config) paths() []string {
	if len(cfg.StreamPaths) > 0 {
		return cfg.StreamPaths
	}
	return []string{defaultStreamPath}
}

type garminDiscovery struct {
	resource.Named
	resource.AlwaysRebuild
	resource.TriviallyCloseable

	cfg    *Config
	logger logging.Logger
}

func newDiscovery(_ context.Context, _ resource.Dependencies,
	conf resource.Config,
	logger logging.Logger,
) (discovery.Service, error) {
	cfg, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return nil, err
	}

	return &garminDiscovery{
		Named:  conf.ResourceName().AsNamed(),
		cfg:    cfg,
		logger: logger,
	}, nil
}

// DiscoverResources browses for Garmin cameras over mDNS and returns a camera
// config for each discovered camera (one per configured stream path).
func (dis *garminDiscovery) DiscoverResources(ctx context.Context, _ map[string]any) ([]resource.Config, error) {
	cameras, err := browseGarmin(ctx, dis.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to browse for garmin cameras: %w", err)
	}

	return camerasToConfigs(cameras, dis.cfg, dis.logger)
}

// camerasToConfigs turns discovered cameras into camera resource configs. Pure
// (no I/O) so it can be unit-tested over ServiceEntry-derived fixtures.
func camerasToConfigs(cameras []Camera, cfg *Config, logger logging.Logger) ([]resource.Config, error) {
	configs := []resource.Config{}
	paths := cfg.paths()
	port := cfg.port()

	for _, cam := range cameras {
		host := cam.addressHost()
		if host == "" {
			logger.Warnf("skipping garmin camera %q: no hostname or IP advertised", cam.Instance)
			continue
		}

		for _, path := range paths {
			address := fmt.Sprintf("rtsp://%s:%d%s", host, port, path)
			name := cameraName(cam.Instance, path, len(paths) > 1)
			cfg, err := createCameraConfig(name, address)
			if err != nil {
				return nil, err
			}
			configs = append(configs, cfg)
		}
	}

	return configs, nil
}

// nonNameChars matches runs of characters not allowed in a Viam resource name.
var nonNameChars = regexp.MustCompile("[^a-zA-Z0-9_-]+")

// cameraName builds a deterministic, sanitized component name from the mDNS
// instance. When a camera yields multiple stream paths, the sanitized path is
// appended to keep names unique and stable across reconfigures.
func cameraName(instance, path string, multiPath bool) string {
	name := strings.Trim(nonNameChars.ReplaceAllString(instance, "-"), "-")
	if multiPath {
		pathSuffix := strings.Trim(nonNameChars.ReplaceAllString(path, "-"), "-")
		name = fmt.Sprintf("%s-%s", name, pathSuffix)
	}
	return name
}

// createCameraConfig builds a camera resource.Config for the given RTSP address.
func createCameraConfig(name, address string) (resource.Config, error) {
	// Use viamrtsp's Config struct so a breaking change surfaces at compile time.
	rtpPassthrough := true
	attributes := viamrtsp.Config{Address: address, RTPPassthrough: &rtpPassthrough}

	jsonBytes, err := json.Marshal(attributes)
	if err != nil {
		return resource.Config{}, err
	}
	var result map[string]interface{}
	if err = json.Unmarshal(jsonBytes, &result); err != nil {
		return resource.Config{}, err
	}

	return resource.Config{
		Name: name, API: camera.API, Model: viamrtsp.ModelAgnostic,
		Attributes: result, ConvertedAttributes: &attributes,
	}, nil
}
