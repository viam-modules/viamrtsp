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

// Model is the model for a upnp discovery service for rtsp cameras.
var (
	Model         = viamrtsp.Family.WithModel("upnp")
	errNoQueries  = errors.New("must provide a query to use the discovery service")
	errEmptyQuery = errors.New("queries cannot be empty")
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
	Queries []queryConfig `json:"queries,omitempty"`
	// UseRootOnly = false will search all services on a network, rather than just the root services.
	// setting this to true is useful when you already know what services/endpoints you wish to use from a device.
	UseRootOnly bool `json:"root_only_search"`
}

type queryConfig struct {
	ModelName    string `json:"model_name"`
	Manufacturer string `json:"manufacturer"`
	SerialNumber string `json:"serial_number"`
	Network      string `json:"network"`
	// users define what endpoints they want to append to discovered queries.
	Endpoints []string `json:"endpoints"`
}

// Validate validates the discovery service.
func (cfg *Config) Validate(_ string) ([]string, []string, error) {
	if len(cfg.Queries) == 0 {
		return []string{}, nil, errNoQueries
	}
	for _, query := range cfg.Queries {
		if query.ModelName == "" && query.Manufacturer == "" && query.SerialNumber == "" {
			return []string{}, nil, errEmptyQuery
		}
	}
	return []string{}, nil, nil
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

	dis.queries, dis.endpointMap = convertqueryConfigToDeviceQuery(cfg.Queries)

	return dis, nil
}

// convertqueryConfigToDeviceQuery pulls out the device query from a queryConfig and creates a map of queries to endpoints.
// The map is needed to add the endpoints to a host when creating the RTSP url.
func convertqueryConfigToDeviceQuery(queryCfgs []queryConfig) ([]viamupnp.DeviceQuery, map[viamupnp.DeviceQuery][]string) {
	var queries []viamupnp.DeviceQuery
	endpointMap := make(map[viamupnp.DeviceQuery][]string)

	for _, queryCfg := range queryCfgs {
		upnpQuery := viamupnp.DeviceQuery{
			ModelName:    queryCfg.ModelName,
			Manufacturer: queryCfg.Manufacturer,
			SerialNumber: queryCfg.SerialNumber,
			Network:      queryCfg.Network,
		}
		queries = append(queries, upnpQuery)
		if len(queryCfg.Endpoints) > 0 {
			endpointMap[upnpQuery] = queryCfg.Endpoints
		}
	}
	return queries, endpointMap
}

// DiscoverResources discovers different rtsp cameras that use upnp.
func (dis *upnpDiscovery) DiscoverResources(ctx context.Context, extra map[string]any) ([]resource.Config, error) {
	cams := []resource.Config{}

	discoverQueries := dis.queries

	extraQuery, ok := getQueryFromExtra(extra)
	if ok {
		discoverQueries = append(discoverQueries, extraQuery)
	}
	// worth noting that the discovered hosts are not guaranteed to be rtsp camera.
	// We assume that the user knew what they were looking for.
	hosts, hostmap, err := viamupnp.FindHost(ctx, dis.logger, discoverQueries, dis.rootOnly)
	if err != nil {
		return nil, err
	}

	for hostNum, host := range hosts {
		query := hostmap[host]
		endpoints, ok := dis.endpointMap[query]
		// the query had no endpoints
		if !ok {
			camConfig, err := createCameraConfig(createCameraName(hostNum, -1, query), "rtsp://"+host, query)
			if err != nil {
				return nil, err
			}
			cams = append(cams, camConfig)
			continue
		}
		for endpointNum, endpoint := range endpoints {
			camConfig, err := createCameraConfig(createCameraName(hostNum, endpointNum, query),
				fmt.Sprintf("rtsp://%s:%s", host, endpoint), query)
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
	if stripManufacturer := reg.ReplaceAllString(query.Manufacturer, ""); stripManufacturer != "" {
		camName = fmt.Sprintf("%s-%s", camName, stripManufacturer)
	}
	if stripModel := reg.ReplaceAllString(query.ModelName, ""); stripModel != "" {
		camName = fmt.Sprintf("%s-%s", camName, stripModel)
	}
	if stripSerial := reg.ReplaceAllString(query.SerialNumber, ""); stripSerial != "" {
		camName = fmt.Sprintf("%s-%s", camName, stripSerial)
	}
	return camName
}

func createCameraConfig(name, address string, query viamupnp.DeviceQuery) (resource.Config, error) {
	// using the camera's Config struct in case a breaking change occurs
	_true := true
	attributes := viamrtsp.Config{Address: address, Query: query, RTPPassthrough: &_true}
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
