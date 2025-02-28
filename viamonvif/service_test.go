// Package viamonvif provides the discovery service that wraps ONVIF integration for the viamrtsp module
package viamonvif

import (
	"context"
	"testing"

	"github.com/viam-modules/viamrtsp"
	"github.com/viam-modules/viamrtsp/viamonvif/device"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/discovery"
	"go.viam.com/test"
)

func TestDiscoveryService(t *testing.T) {
	cfg := Config{}
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
	conf, err := createCameraConfig(camName, "", camURL)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, conf.Name, test.ShouldEqual, camName)
	cfg, err := resource.NativeConfig[*viamrtsp.Config](conf)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, cfg.Address, test.ShouldEqual, camURL)
	test.That(t, *cfg.RTPPassthrough, test.ShouldBeTrue)
	test.That(t, cfg.DiscoveryDep, test.ShouldEqual, "")

	discSvcDep := "discovery-service-dependency"
	conf, err = createCameraConfig(camName, discSvcDep, camURL)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, conf.Name, test.ShouldEqual, camName)
	cfg, err = resource.NativeConfig[*viamrtsp.Config](conf)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, cfg.Address, test.ShouldEqual, camURL)
	test.That(t, *cfg.RTPPassthrough, test.ShouldBeTrue)
	test.That(t, cfg.DiscoveryDep, test.ShouldEqual, discSvcDep)
}

func TestDiscoveryConfig(t *testing.T) {
	t.Run("Test Empty Config", func(t *testing.T) {
		cfg := Config{}
		deps, err := cfg.Validate("")
		test.That(t, err, test.ShouldBeNil)
		test.That(t, deps, test.ShouldBeEmpty)
	})
	t.Run("Test Valid config", func(t *testing.T) {
		cfg := Config{Credentials: []device.Credentials{
			{User: "user1", Pass: "pass1"},
			{User: "user2", Pass: "pass2"},
			{User: "user3", Pass: ""},
			{User: "", Pass: ""},
		}}
		deps, err := cfg.Validate("")
		test.That(t, err, test.ShouldBeNil)
		test.That(t, deps, test.ShouldBeEmpty)
	})
	t.Run("Test Invalid Config", func(t *testing.T) {
		cfg := Config{Credentials: []device.Credentials{{User: "", Pass: "pass1"}}}
		deps, err := cfg.Validate("")
		test.That(t, err.Error(), test.ShouldContainSubstring, "credential missing username, has password pass1")
		test.That(t, deps, test.ShouldBeEmpty)
	})
}

func TestGetCredFromExtra(t *testing.T) {
	t.Run("Test good extra with User and Pass as strings", func(t *testing.T) {
		extra := map[string]any{
			"User": "user",
			"Pass": "pass",
		}
		cred, ok := getCredFromExtra(extra)
		test.That(t, cred.User, test.ShouldEqual, "user")
		test.That(t, cred.Pass, test.ShouldEqual, "pass")
		test.That(t, ok, test.ShouldBeTrue)
	})
	t.Run("Test good extra with no Pass", func(t *testing.T) {
		extra := map[string]any{
			"User": "user",
		}
		cred, ok := getCredFromExtra(extra)
		test.That(t, cred.User, test.ShouldEqual, "user")
		test.That(t, cred.Pass, test.ShouldEqual, "")
		test.That(t, ok, test.ShouldBeTrue)
	})
	t.Run("Test bad extra with no strings", func(t *testing.T) {
		extra := map[string]any{
			"User": 1,
			"Pass": true,
		}
		cred, ok := getCredFromExtra(extra)
		test.That(t, cred.User, test.ShouldEqual, "")
		test.That(t, cred.Pass, test.ShouldEqual, "")
		test.That(t, ok, test.ShouldBeFalse)
	})
	t.Run("Test nil cred", func(t *testing.T) {
		cred, ok := getCredFromExtra(nil)
		test.That(t, cred.User, test.ShouldEqual, "")
		test.That(t, cred.Pass, test.ShouldEqual, "")
		test.That(t, ok, test.ShouldBeFalse)
	})
}
