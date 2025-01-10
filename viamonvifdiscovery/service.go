// Package viamonvifdiscovery provides the discovery service that wraps ONVIF integration for the viamrtsp module
package viamonvifdiscovery

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/viam-modules/viamrtsp"
	"github.com/viam-modules/viamrtsp/viamonvif"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/discovery"
)

// Model is the model for a rtsp discovery service.
var Model = viamrtsp.Family.WithModel("discovery")

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

// Creds are the login credentials that a user can input.
type Creds struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	credNumber int
}

// Config is the config for the discovery service.
type Config struct {
	Credentials []Creds `json:"credentials"`
}

// Validate validates the discovery service.
func (cfg *Config) Validate(_ string) ([]string, error) {
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

func newDiscovery(_ context.Context,
	conf resource.Config,
	logger logging.Logger,
) (discovery.Service, error) {
	cfg, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return nil, err
	}
	dis := &rtspDiscovery{
		Named:       conf.ResourceName().AsNamed(),
		Credentials: cfg.Credentials,
		logger:      logger,
	}

	dis.setCredNumbers()

	// we always want to serve insecure cameras
	// dis.Credentials = append(dis.Credentials, emptyCred)
	return dis, nil
}

func (dis *rtspDiscovery) setCredNumbers() {
	// usernames do not have to be unique, so we want to add an additional label to ensure cameras have unique names.
	for index, cred := range dis.Credentials {
		for counter, otherCreds := range dis.Credentials {
			if counter <= index {
				continue
			}
			// increase the credNumber for later cameras.
			if cred.Username == otherCreds.Username {
				dis.Credentials[counter].credNumber += 1
			}
		}
		dis.logger.Info("yo cred numbers: ", dis.Credentials[index].credNumber)
	}
}

// Close closes the discovery service.
func (dis *rtspDiscovery) Close(_ context.Context) error {
	return nil
}

// DiscoverResources discovers different rtsp cameras that use onvif.
func (dis *rtspDiscovery) DiscoverResources(_ context.Context, _ map[string]any) ([]resource.Config, error) {
	potentialCams := []resource.Config{}
	// test insecure creds first
	list, err := viamonvif.DiscoverCameras(emptyCred.Username, emptyCred.Password, dis.logger, nil)
	if err != nil {
		return nil, err
	}
	insecureURLs := []string{}
	for cameraNumber, l := range list.Cameras {
		dis.logger.Debugf("%s %s %s", l.Manufacturer, l.Model, l.SerialNumber)
		insecureURLs = append(insecureURLs, l.NoLoginURLs...)

		for _, u := range l.RTSPURLs {
			dis.logger.Debugf("\t%s", u)
			cfg, err := createCameraConfig(emptyCred.createName(cameraNumber), u)
			if err != nil {
				return nil, err
			}
			potentialCams = append(potentialCams, cfg)
		}
	}

	for _, cred := range dis.Credentials {
		list, err := viamonvif.DiscoverCameras(cred.Username, cred.Password, dis.logger, nil)
		if err != nil {
			return nil, err
		}

		for camera_number, l := range list.Cameras {
			dis.logger.Debugf("%s %s %s", l.Manufacturer, l.Model, l.SerialNumber)
			for index, u := range l.RTSPURLs {
				// // skip over cameras that were already discovered with insecure creds
				if slices.Contains(insecureURLs, l.NoLoginURLs[index]) {
					continue
				}
				dis.logger.Debugf("\t%s", u)
				cfg, err := createCameraConfig(cred.createName(camera_number), u)
				if err != nil {
					return nil, err
				}
				potentialCams = append(potentialCams, cfg)
			}
		}
	}
	return potentialCams, nil
}

// set the camera name based on the Username and camera number.
func (cred *Creds) createName(index int) string {
	if cred.Username == "" {
		return fmt.Sprintf("Camera_Insecure_%v", index)
	}
	if cred.credNumber > 0 {
		fmt.Println("yo credNumber")
		return fmt.Sprintf("Camera_%s-%v_%v", cred.Username, cred.credNumber, index)
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
