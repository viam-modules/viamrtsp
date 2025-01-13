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
			Constructor: newDiscovery,
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
		Credentials: append([]Creds{emptyCred}, cfg.Credentials...),
		logger:      logger,
	}

	dis.setCredNumbers()
	return dis, nil
}

// since camera logins can have the same username,
// check how many repeated usernames we have and index them, to ensure cameras have unique names.
func (dis *rtspDiscovery) setCredNumbers() {
	// usernames do not have to be unique, so we want to add an additional label to ensure cameras have unique names.
	for index, cred := range dis.Credentials {
		for counter, otherCreds := range dis.Credentials {
			if counter <= index {
				continue
			}
			// increase the credNumber for later cameras.
			if cred.Username == otherCreds.Username {
				dis.Credentials[counter].credNumber++
			}
		}
	}
}

// Close closes the discovery service.
func (dis *rtspDiscovery) Close(_ context.Context) error {
	return nil
}

// DiscoverResources discovers different rtsp cameras that use onvif.
func (dis *rtspDiscovery) DiscoverResources(_ context.Context, _ map[string]any) ([]resource.Config, error) {
	potentialCams := []resource.Config{}
	insecureURLs := []string{}

	for _, cred := range dis.Credentials {
		// if we have multiple empty/insecure credentials configured for some reason, skip to the next credential.
		// only skip if we have not processed insecure credentials yet.
		if cred.isInsecure() && len(insecureURLs) > 0 {
			continue
		}

		list, err := viamonvif.DiscoverCameras(cred.Username, cred.Password, dis.logger, nil)
		if err != nil {
			return nil, err
		}
		cameraNumber := 0
		for _, l := range list.Cameras {
			dis.logger.Debugf("%s %s %s", l.Manufacturer, l.Model, l.SerialNumber)
			// some cameras return with no urls. explicitly skipping those so the behavior is clear in the service.
			if len(l.RTSPURLs) == 0 {
				dis.logger.Debugf("No urls found for camera, skipping. %s %s %s", l.Manufacturer, l.Model, l.SerialNumber)
				continue
			}
			cameraNumber++
			camConfigs, err := createCamerasFromURLs(cred.createName(cameraNumber), l, insecureURLs, dis.logger)
			if err != nil {
				return nil, err
			}
			potentialCams = append(potentialCams, camConfigs...)

			// if we are processing insecure creds, record the urls so we do not duplicate the cameras when processing cameras with creds.
			if cred.isInsecure() {
				insecureURLs = append(insecureURLs, l.NoLoginURLs...)
			}
		}
	}
	return potentialCams, nil
}

func createCamerasFromURLs(cameraName string, l viamonvif.CameraInfo, insecureCams []string,
	logger logging.Logger,
) ([]resource.Config, error) {
	potentialCams := []resource.Config{}
	for index, u := range l.RTSPURLs {
		// skip over cameras that were already discovered with insecure creds
		if slices.Contains(insecureCams, l.NoLoginURLs[index]) {
			continue
		}
		logger.Debugf("\t%s", u)
		cfg, err := createCameraConfig(cameraName, u)
		if err != nil {
			return nil, err
		}
		potentialCams = append(potentialCams, cfg)
	}
	return potentialCams, nil
}

// set the camera name based on the Username and camera number.
func (cred *Creds) createName(index int) string {
	if cred.Username == "" {
		return fmt.Sprintf("Camera_Insecure_%v", index)
	}
	if cred.credNumber > 0 {
		return fmt.Sprintf("Camera_%s-%v_%v", cred.Username, cred.credNumber, index)
	}
	return fmt.Sprintf("Camera_%s_%v", cred.Username, index)
}

func (cred *Creds) isInsecure() bool {
	return cred.Username == "" && cred.Password == ""
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

	return resource.Config{
		Name: name, API: camera.API, Model: viamrtsp.ModelAgnostic,
		Attributes: result, ConvertedAttributes: &attributes,
	}, nil
}
