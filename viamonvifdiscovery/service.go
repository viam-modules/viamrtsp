// Package viamonvifdiscovery provides the discovery service that wraps ONVIF integration for the viamrtsp module
package viamonvifdiscovery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/viam-modules/viamrtsp"
	"github.com/viam-modules/viamrtsp/viamonvif"
	"github.com/viam-modules/viamrtsp/viamonvif/device"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/discovery"
)

// Model is the model for a rtsp discovery service.
var Model = viamrtsp.Family.WithModel("onvif")

func init() {
	resource.RegisterService(
		discovery.API,
		Model,
		resource.Registration[discovery.Service, *Config]{
			Constructor: newDiscovery,
		})
}

// Config is the config for the discovery service.
type Config struct {
	Credentials []device.Credentials `json:"credentials"`
}

// Validate validates the discovery service.
func (cfg *Config) Validate(_ string) ([]string, error) {
	// check that all creds have both usernames and passwords set. Note a credential can have both fields empty
	for _, cred := range cfg.Credentials {
		if cred.Pass != "" && cred.User == "" {
			return nil, fmt.Errorf("credential missing username, has password %v", cred.Pass)
		}
	}
	return []string{}, nil
}

type rtspDiscovery struct {
	resource.Named
	resource.AlwaysRebuild
        resource.TriviallyCloseable
	Credentials []device.Credentials
	logger      logging.Logger
}

func newDiscovery(_ context.Context, _ resource.Dependencies,
	conf resource.Config,
	logger logging.Logger,
) (discovery.Service, error) {
	cfg, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return nil, err
	}
	dis := &rtspDiscovery{
		Named: conf.ResourceName().AsNamed(),
		// place the empty credentials first, so we find the insecure cameras before other cameras.
		Credentials: cfg.Credentials,
		logger:      logger,
	}

	return dis, nil
}


// DiscoverResources discovers different rtsp cameras that use onvif.
func (dis *rtspDiscovery) DiscoverResources(ctx context.Context, _ map[string]any) ([]resource.Config, error) {
	potentialCams := []resource.Config{}

	list, err := viamonvif.DiscoverCameras(ctx, dis.Credentials, nil, dis.logger)
	if err != nil {
		return nil, err
	}

	for _, l := range list.Cameras {
		dis.logger.Debugf("%s %s %s", l.Manufacturer, l.Model, l.SerialNumber)
		// some cameras return with no urls. explicitly skipping those so the behavior is clear in the service.
		if len(l.RTSPURLs) == 0 {
			dis.logger.Errorf("No urls found for camera, skipping. %s %s %s", l.Manufacturer, l.Model, l.SerialNumber)
			continue
		}
		camConfigs, err := createCamerasFromURLs(l, dis.logger)
		if err != nil {
			return nil, err
		}
		potentialCams = append(potentialCams, camConfigs...)
	}
	return potentialCams, nil
}

func createCamerasFromURLs(l viamonvif.CameraInfo, logger logging.Logger) ([]resource.Config, error) {
	potentialCams := []resource.Config{}
	for _, u := range l.RTSPURLs {
		logger.Debugf("camera URL:\t%s", u)
		cfg, err := createCameraConfig(l.Name(), u)
		if err != nil {
			return nil, err
		}
		potentialCams = append(potentialCams, cfg)
	}
	return potentialCams, nil
}

func createCameraConfig(name, address string) (resource.Config, error) {
	// using the camera's Config struct in case a breaking change occurs
	attributes := viamrtsp.Config{Address: address}
	var result map[string]interface{}

	// marshal to bytes
	jsonBytes, err := json.Marshal(attributes)
	if err != nil {
		return resource.Config{}, err
	}

	// convert to map to be used as attributes in resource.Config
	if err = json.Unmarshal(jsonBytes, &result); err != nil {
		return resource.Config{}, err
	}

	return resource.Config{
		Name: name, API: camera.API, Model: viamrtsp.ModelAgnostic,
		Attributes: result, ConvertedAttributes: &attributes,
	}, nil
}
