// Package upnpdiscovery provides the discovery service that wraps UPnP integration for the viamrtsp module
package upnpdiscovery

import (
	"context"
	"testing"

	"github.com/erh/viamupnp"
	"github.com/viam-modules/viamrtsp"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/discovery"
	"go.viam.com/test"
)

func TestDiscoveryService(t *testing.T) {
	deviceQuery := viamupnp.DeviceQuery{ModelName: "bad"}
	cfg := Config{Queries: []queryConfig{{DeviceQuery: deviceQuery}}}
	ctx := context.Background()
	logger := logging.NewTestLogger(t)

	t.Run("Test Default Service with no cameras", func(t *testing.T) {
		testName := "test"
		resourceCfg := resource.Config{API: discovery.API, Model: Model, Name: testName, ConvertedAttributes: &cfg}
		dis, err := newDiscovery(ctx, nil, resourceCfg, logger)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, dis, test.ShouldNotBeNil)
		test.That(t, dis.Name().ShortName(), test.ShouldResemble, testName)
		cfgs, err := dis.DiscoverResources(ctx, nil)
		test.That(t, cfgs, test.ShouldBeEmpty)
		test.That(t, err, test.ShouldBeError, errNoCamerasFound)
	})
}

func TestCamConfig(t *testing.T) {
	camName := "my-cam"
	camURL := "my-cam-url"
	conf, err := createCameraConfig(camName, camURL)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, conf.Name, test.ShouldEqual, camName)
	cfg, err := resource.NativeConfig[*viamrtsp.Config](conf)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, cfg.Address, test.ShouldEqual, camURL)
	test.That(t, *cfg.RTPPassthrough, test.ShouldBeTrue)
}

func TestDiscoveryConfig(t *testing.T) {
	t.Run("Test Empty Config", func(t *testing.T) {
		cfg := Config{}
		deps, err := cfg.Validate("")
		test.That(t, err, test.ShouldBeError, errNoQueries)
		test.That(t, deps, test.ShouldBeEmpty)
	})
	t.Run("Test Valid config", func(t *testing.T) {
		deviceQuery := viamupnp.DeviceQuery{ModelName: "good"}
		cfg := Config{Queries: []queryConfig{{DeviceQuery: deviceQuery}}}

		deps, err := cfg.Validate("")
		test.That(t, err, test.ShouldBeNil)
		test.That(t, deps, test.ShouldBeEmpty)
	})
	t.Run("Test Invalid Config", func(t *testing.T) {
		deviceQuery := viamupnp.DeviceQuery{Network: "bad"}
		cfg := Config{Queries: []queryConfig{{DeviceQuery: deviceQuery}}}
		deps, err := cfg.Validate("")
		test.That(t, err.Error(), test.ShouldBeError, errEmptyQuery)
		test.That(t, deps, test.ShouldBeEmpty)
	})
}

func TestGetQueryFromExtra(t *testing.T) {
	t.Run("Test good extra with full query as strings", func(t *testing.T) {
		extra := map[string]any{
			"model_name":    "model",
			"manufacturer":  "manufacturer",
			"serial_number": "serial_number",
			"network":       "network",
		}
		query, ok := getQueryFromExtra(extra)
		test.That(t, query.ModelName, test.ShouldEqual, "model")
		test.That(t, query.Manufacturer, test.ShouldEqual, "manufacturer")
		test.That(t, query.SerialNumber, test.ShouldEqual, "serial_number")
		test.That(t, query.Network, test.ShouldEqual, "network")
		test.That(t, ok, test.ShouldBeTrue)
	})
	t.Run("Test good extra with only one field defined correctly", func(t *testing.T) {
		// we currently check that the extra params exist and are strings at the same time, so stuff like this works.
		extra := map[string]any{
			"model_name":   "model",
			"manufacturer": 2.0,
		}
		query, ok := getQueryFromExtra(extra)
		test.That(t, query.ModelName, test.ShouldEqual, "model")
		test.That(t, query.Manufacturer, test.ShouldEqual, "")
		test.That(t, query.SerialNumber, test.ShouldEqual, "")
		test.That(t, query.Network, test.ShouldEqual, "")
		test.That(t, ok, test.ShouldBeTrue)
	})
	t.Run("Test bad extra with no strings", func(t *testing.T) {
		extra := map[string]any{
			"model_name":   1,
			"manufacturer": 2.0,
		}
		query, ok := getQueryFromExtra(extra)
		test.That(t, query.ModelName, test.ShouldEqual, "")
		test.That(t, query.Manufacturer, test.ShouldEqual, "")
		test.That(t, query.SerialNumber, test.ShouldEqual, "")
		test.That(t, query.Network, test.ShouldEqual, "")
		test.That(t, ok, test.ShouldBeFalse)
	})
	t.Run("Test nil cred", func(t *testing.T) {
		query, ok := getQueryFromExtra(nil)
		test.That(t, query.ModelName, test.ShouldEqual, "")
		test.That(t, query.Manufacturer, test.ShouldEqual, "")
		test.That(t, query.SerialNumber, test.ShouldEqual, "")
		test.That(t, query.Network, test.ShouldEqual, "")
		test.That(t, ok, test.ShouldBeFalse)
	})
}
