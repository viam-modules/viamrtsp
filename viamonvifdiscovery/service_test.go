// Package viamonvifdiscovery provides the discovery service that wraps ONVIF integration for the viamrtsp module
package viamonvifdiscovery

import (
	"context"
	"testing"

	"github.com/viam-modules/viamrtsp"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/discovery"
	"go.viam.com/test"
)

func TestDiscoveryService(t *testing.T) {
	cfg := Config{}
	ctx := context.Background()
	logger := logging.NewTestLogger(t)

	t.Run("Test Default Service", func(t *testing.T) {
		testName := "test"
		resourceCfg := resource.Config{API: discovery.API, Model: Model, Name: testName, ConvertedAttributes: &cfg}
		dis, err := newDiscovery(ctx, nil, resourceCfg, logger)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, dis, test.ShouldNotBeNil)
		test.That(t, dis.Name().ShortName(), test.ShouldResemble, testName)
		test.That(t, dis.Close(ctx), test.ShouldBeNil)
	})
}

func TestCredentials(t *testing.T) {
	creds := []Creds{{Username: "", Password: ""}, {Username: "user1", Password: "pass1"}, {Username: "user2", Password: "pass2"},
		{Username: "user1", Password: "pass3"}, {Username: "user1", Password: "pass4"}, {Username: "user2", Password: "pass5"}}
	dis := rtspDiscovery{Credentials: creds}
	dis.setCredNumbers()
	test.That(t, dis.Credentials[0].credNumber, test.ShouldEqual, 0)
	test.That(t, dis.Credentials[1].credNumber, test.ShouldEqual, 0)
	test.That(t, dis.Credentials[2].credNumber, test.ShouldEqual, 0)
	test.That(t, dis.Credentials[3].credNumber, test.ShouldEqual, 1)
	test.That(t, dis.Credentials[4].credNumber, test.ShouldEqual, 2)
	test.That(t, dis.Credentials[5].credNumber, test.ShouldEqual, 1)
	dis.Credentials[0].createName(0)
	test.That(t, dis.Credentials[0].createName(6), test.ShouldEqual, "Camera_Insecure_6")
	test.That(t, dis.Credentials[1].createName(7), test.ShouldEqual, "Camera_user1_7")
	test.That(t, dis.Credentials[2].createName(8), test.ShouldEqual, "Camera_user2_8")
	test.That(t, dis.Credentials[3].createName(9), test.ShouldEqual, "Camera_user1-1_9")
	test.That(t, dis.Credentials[4].createName(10), test.ShouldEqual, "Camera_user1-2_10")
	test.That(t, dis.Credentials[5].createName(11), test.ShouldEqual, "Camera_user2-1_11")
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
}

func TestDiscoveryConfig(t *testing.T) {
	t.Run("Test Empty Config", func(t *testing.T) {
		cfg := Config{}
		deps, err := cfg.Validate("")
		test.That(t, err, test.ShouldBeNil)
		test.That(t, deps, test.ShouldBeEmpty)

	})
	t.Run("Test Valid config", func(t *testing.T) {
		cfg := Config{Credentials: []Creds{{Username: "user1", Password: "pass1"}, {Username: "user2", Password: "pass2"},
			{Username: "user3", Password: "pass3"}, {Username: "", Password: ""}}}
		deps, err := cfg.Validate("")
		test.That(t, err, test.ShouldBeNil)
		test.That(t, deps, test.ShouldBeEmpty)
	})
	t.Run("Test Invalid Config", func(t *testing.T) {
		cfg := Config{Credentials: []Creds{{Username: "user1", Password: ""}}}
		deps, err := cfg.Validate("")
		test.That(t, err.Error(), test.ShouldContainSubstring, "credential user1 missing password")
		test.That(t, deps, test.ShouldBeEmpty)
		cfg2 := Config{Credentials: []Creds{{Username: "", Password: "pass1"}}}
		deps, err = cfg2.Validate("")
		test.That(t, err.Error(), test.ShouldContainSubstring, "credential missing username, has password pass1")
		test.That(t, deps, test.ShouldBeEmpty)
	})
}
