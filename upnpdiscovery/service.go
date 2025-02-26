// Package upnpdiscovery provides the discovery service that wraps UPnP integration for the viamrtsp module
package upnpdiscovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/erh/viamupnp"
	"github.com/viam-modules/viamrtsp"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/discovery"
)

// Model is the model for a rtsp discovery service.
var (
	Model             = viamrtsp.Family.WithModel("upnp")
	errNoCamerasFound = errors.New("no cameras found, ensure cameras are working or check queries")
	errNoQueries      = errors.New("must provide a query to use the discovery service")
	errEmptyQuery     = errors.New("queries cannot be empty")
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
	Queries     []QueryConfig `json:"queries,omitempty"`
	UseRootOnly bool          `json:"root_only_search"`
}

type QueryConfig struct {
	viamupnp.DeviceQuery
	Endpoints []string `json:"endpoints"`
}

// Validate validates the discovery service.
func (cfg *Config) Validate(_ string) ([]string, error) {
	if len(cfg.Queries) == 0 {
		return []string{}, errNoQueries
	}
	for _, query := range cfg.Queries {
		if query.ModelName == "" && query.Manufacturer == "" && query.SerialNumber == "" {
			return []string{}, errEmptyQuery
		}
	}
	return []string{}, nil
}

type upnpDiscovery struct {
	resource.Named
	resource.AlwaysRebuild
	resource.TriviallyCloseable
	queries     []viamupnp.DeviceQuery
	endpointMap map[viamupnp.DeviceQuery][]string
	logger      logging.Logger
	rootOnly    bool
}

func newDiscovery(_ context.Context, _ resource.Dependencies,
	conf resource.Config,
	logger logging.Logger,
) (discovery.Service, error) {
	cfg, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return nil, err
	}

	dis := &upnpDiscovery{
		Named:    conf.ResourceName().AsNamed(),
		rootOnly: cfg.UseRootOnly,
		logger:   logger,
	}

	dis.queries, dis.endpointMap = convertQueryConfigToDeviceQuery(cfg.Queries)

	return dis, nil
}

// convertQueryConfigToDeviceQuery pulls out the device query from a QueryConfig and creates a map of queries to endpoints.
// The map is needed to add the endpoints to a host when creating the RTSP url.
func convertQueryConfigToDeviceQuery(queryCfgs []QueryConfig) ([]viamupnp.DeviceQuery, map[viamupnp.DeviceQuery][]string) {
	var queries []viamupnp.DeviceQuery
	endpointMap := make(map[viamupnp.DeviceQuery][]string)

	for _, queryCfg := range queryCfgs {
		queries = append(queries, queryCfg.DeviceQuery)
		if len(queryCfg.Endpoints) > 0 {
			endpointMap[queryCfg.DeviceQuery] = queryCfg.Endpoints
		}
	}
	return queries, endpointMap
}

// DiscoverResources discovers different rtsp cameras that use onvif.
func (dis *upnpDiscovery) DiscoverResources(ctx context.Context, extra map[string]any) ([]resource.Config, error) {
	cams := []resource.Config{}

	discoverQueries := dis.queries

	extraQuery, ok := getQueryFromExtra(extra)
	if ok {
		discoverQueries = append(discoverQueries, extraQuery)
	}
	hosts, hostmap, err := viamupnp.FindHost(ctx, dis.logger, discoverQueries, dis.rootOnly)
	if err != nil {
		return nil, err
	}
	if len(hosts) == 0 {
		return nil, errNoCamerasFound
	}
	for hostNum, host := range hosts {
		query := hostmap[host]
		endpoints, ok := dis.endpointMap[query]
		// the query had no endpoints
		if !ok {
			camConfig, err := createCameraConfig(createCameraName(hostNum, -1, query), host)
			if err != nil {
				return nil, err
			}
			cams = append(cams, camConfig)
			continue
		}
		for endpointNum, endpoint := range endpoints {
			camConfig, err := createCameraConfig(createCameraName(hostNum, endpointNum, query), fmt.Sprintf("%s:%s", host, endpoint))
			if err != nil {
				return nil, err
			}
			cams = append(cams, camConfig)
		}
	}

	return cams, nil
}

// regex to remove non alpha numerics.
var reg = regexp.MustCompile("[^a-zA-Z0-9]+")

// createCameraName creates a camera name based on the query.
func createCameraName(hostNum, endpointNum int, query viamupnp.DeviceQuery) string {
	camName := fmt.Sprintf("camera%v", hostNum)
	if endpointNum != -1 {
		camName = fmt.Sprintf("%s-endpoint%v", camName, endpointNum)
	}
	if stripManufacturer := reg.ReplaceAllString(query.Manufacturer, ""); stripManufacturer == "" {
		camName = fmt.Sprintf("%s-%s", camName, stripManufacturer)
	}
	if stripModel := reg.ReplaceAllString(query.ModelName, ""); stripModel == "" {
		camName = fmt.Sprintf("%s-%s", camName, stripModel)
	}
	if stripSerial := reg.ReplaceAllString(query.SerialNumber, ""); stripSerial == "" {
		camName = fmt.Sprintf("%s-%s", camName, stripSerial)
	}
	return camName
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

func getQueryFromExtra(extra map[string]any) (viamupnp.DeviceQuery, bool) {
	// check for a username from extras
	extraModel, ok := extra["model_name"].(string)
	if !ok {
		extraModel = ""
	}
	// not requiring a password to match config
	extraManufacturer, ok := extra["manufacturer"].(string)
	if !ok {
		extraManufacturer = ""
	}
	// not requiring a password to match config
	extraSerial, ok := extra["serial_number"].(string)
	if !ok {
		extraSerial = ""
	}
	// not requiring a password to match config
	extraNetwork, ok := extra["network"].(string)
	if !ok {
		extraNetwork = ""
	}
	if extraModel == "" && extraManufacturer == "" && extraSerial == "" {
		return viamupnp.DeviceQuery{}, false
	}

	return viamupnp.DeviceQuery{
		ModelName: extraModel, Manufacturer: extraManufacturer,
		SerialNumber: extraSerial, Network: extraNetwork,
	}, true
}
