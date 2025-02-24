// Package viamonvif provides the discovery service that wraps ONVIF integration for the viamrtsp module
package viamonvif

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/viam-modules/viamrtsp"
	"github.com/viam-modules/viamrtsp/viamonvif/device"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/discovery"
)

// Model is the model for a rtsp discovery service.
var (
	Model             = viamrtsp.Family.WithModel("onvif")
	errNoCamerasFound = errors.New("no cameras found, ensure cameras are working or check credentials")
	emptyCred         = device.Credentials{}
)

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
	// check that all creds have usernames set. Note a credential can have both fields empty
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
	Credentials []device.Credentials
	mdnsServer  mdnsServer
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
		Named:       conf.ResourceName().AsNamed(),
		Credentials: append([]device.Credentials{emptyCred}, cfg.Credentials...),
		mdnsServer:  newMDNSServer(logger.Sublogger("mdns")),
		logger:      logger,
	}

	return dis, nil
}

// DiscoverResources discovers different rtsp cameras that use onvif.
func (dis *rtspDiscovery) DiscoverResources(ctx context.Context, extra map[string]any) ([]resource.Config, error) {
	cams := []resource.Config{}

	discoverCreds := dis.Credentials

	extraCred, ok := getCredFromExtra(extra)
	if ok {
		discoverCreds = append(discoverCreds, extraCred)
	}
	list, err := DiscoverCameras(ctx, discoverCreds, nil, dis.logger)
	if err != nil {
		return nil, err
	}
	if len(list.Cameras) == 0 {
		return nil, errors.New("no cameras found, ensure cameras are working or check credentials")
	}

	for _, camInfo := range list.Cameras {
		dis.logger.Debugf("%s %s %s", camInfo.Manufacturer, camInfo.Model, camInfo.SerialNumber)
		// some cameras return with no urls. explicitly skipping those so the behavior is clear in the service.
		if len(camInfo.RTSPURLs) == 0 {
			dis.logger.Errorf("No urls found for camera, skipping. %s %s %s",
				camInfo.Manufacturer, camInfo.Model, camInfo.SerialNumber)
			continue
		}

		// tryMDNS will attempt to register an mdns entry for the camera. If successfully
		// registered, `tryMDNS` will additionally mutate the `camInfo.RTSPURLs` to use the dns
		// hostname rather than a raw IP. Such that the camera configs we are about to generate will
		// use the dns hostname.
		camInfo.tryMDNS(&dis.mdnsServer, dis.logger)

		camConfigs, err := createCamerasFromURLs(camInfo, dis.logger)
		if err != nil {
			return nil, err
		}
		cams = append(cams, camConfigs...)
	}

	return cams, nil
}

func (dis *rtspDiscovery) Close(ctx context.Context) error {
	dis.mdnsServer.Shutdown()
	return nil
}

func createCamerasFromURLs(l CameraInfo, logger logging.Logger) ([]resource.Config, error) {
	cams := []resource.Config{}
	for index, u := range l.RTSPURLs {
		logger.Debugf("camera URL:\t%s", u)
		cfg, err := createCameraConfig(l.Name(index), u)
		if err != nil {
			return nil, err
		}
		cams = append(cams, cfg)
	}
	return cams, nil
}

func createCameraConfig(name, address string) (resource.Config, error) {
	// using the camera's Config struct in case a breaking change occurs
	_true := true
	attributes := viamrtsp.Config{Address: address, RTPPassthrough: &_true}
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

func getCredFromExtra(extra map[string]any) (device.Credentials, bool) {
	// check for a username from extras
	extraUser, ok := extra["User"].(string)
	if !ok {
		return device.Credentials{}, false
	}
	// not requiring a password to match config
	extraPass, ok := extra["Pass"].(string)
	if !ok {
		extraPass = ""
	}

	return device.Credentials{User: extraUser, Pass: extraPass}, true
}
