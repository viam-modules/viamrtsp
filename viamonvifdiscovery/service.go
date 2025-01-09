// Package viamonvifdiscovery provides the discovery service that wraps ONVIF integration for the viamrtsp module
package viamonvifdiscovery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/viam-modules/viamrtsp"
	"github.com/viam-modules/viamrtsp/viamonvif"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/discovery"
)

// Model is the model for a rtsp discovery service.
var Model = viamrtsp.Family.WithModel("discovery")

// defining the family in here to avoid circular dependencies while we setup the new discovery method
// var Model = resource.NewModel("viam", "viamrtsp", "discovery")

var emptyCred = Creds{Username: "", Password: ""}

func init() {
	resource.RegisterService(
		discovery.API,
		Model,
		resource.Registration[discovery.Service, *Config]{
			Constructor: func(
				ctx context.Context,
				_ resource.Dependencies,
				conf resource.Config,
				logger logging.Logger,
			) (discovery.Service, error) {
				return newDiscovery(ctx, conf, logger)
			},
		})
}

type Creds struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type Config struct {
	Credentials []Creds `json:"credentials"`
}

func (cfg *Config) Validate(deps string) ([]string, error) {
	// check that all creds have both usernames and passwords set. Note a credential can have both fields empty
	for _, cred := range cfg.Credentials {
		if cred.Username != "" && cred.Password == "" {
			return nil, fmt.Errorf("credential %v missing password", cred.Username)
		}
		if cred.Password != "" && cred.Username == "" {
			return nil, fmt.Errorf("credential missing username, has password %v", cred.Password)
		}
	}
	return nil, nil
}

type rtspDiscovery struct {
	resource.Named
	resource.AlwaysRebuild
	Credentials []Creds
	logger      logging.Logger
}

func newDiscovery(ctx context.Context,
	conf resource.Config,
	logger logging.Logger) (discovery.Service, error) {
	cfg, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return nil, err
	}
	dis := &rtspDiscovery{
		Named:       conf.ResourceName().AsNamed(),
		Credentials: cfg.Credentials,
		logger:      logger,
	}
	dis.Credentials = append(dis.Credentials, emptyCred)
	return dis, nil
}

func (dis *rtspDiscovery) Close(context.Context) error {
	return nil
}

func (dis *rtspDiscovery) DiscoverResources(ctx context.Context, extra map[string]any) ([]resource.Config, error) {
	potentialCams := []resource.Config{}
	for _, cred := range dis.Credentials {
		list, err := viamonvif.DiscoverCameras(cred.Username, cred.Password, dis.logger, nil)
		if err != nil {
			return nil, err
		}

		for index, l := range list.Cameras {
			dis.logger.Debugf("%s %s %s", l.Manufacturer, l.Model, l.SerialNumber)
			for _, u := range l.RTSPURLs {
				dis.logger.Debugf("\t%s", u)
				cfg, err := createCameraConfig(cred.createName(index), u)
				if err != nil {
					return nil, err
				}
				potentialCams = append(potentialCams, cfg)
			}
		}
	}
	return potentialCams, nil
}

func (cred *Creds) createName(index int) string {
	if cred.Username == "" {
		return fmt.Sprintf("Camera_Insecure_%v", index)
	}
	return fmt.Sprintf("Camera_%s_%v", cred.Username, index)
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
	err = json.Unmarshal(jsonBytes, &result)
	if err != nil {
		return resource.Config{}, err
	}
	return resource.Config{Name: name, API: camera.API, Model: viamrtsp.ModelAgnostic, Attributes: result}, nil
}
