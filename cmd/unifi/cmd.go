// Package main is a binary for testing Unifi NVR camera discovery
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/viam-modules/viamrtsp/unifi"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
)

func main() {
	err := realMain()
	if err != nil {
		panic(err)
	}
}

func realMain() error {
	logger := logging.NewLogger("unifi-discovery")

	debug := true
	nvrIP := ""
	token := ""

	flag.BoolVar(&debug, "debug", debug, "enable debug logging")
	flag.StringVar(&nvrIP, "nvr", nvrIP, "NVR IP address (required)")
	flag.StringVar(&token, "token", token, "Unifi API token (required)")

	flag.Parse()

	if debug {
		logger.SetLevel(logging.DEBUG)
	}

	if nvrIP == "" {
		return errors.New("nvr IP address is required (-nvr)")
	}
	if token == "" {
		return errors.New("unifi API token is required (-token)")
	}

	logger.Infof("Testing Unifi discovery with NVR: %s", nvrIP)

	cfg := resource.Config{
		Name: "test-unifi-discovery",
		ConvertedAttributes: &unifi.Config{
			NVRAddress: nvrIP,
			UnifiToken: token,
		},
	}

	svc, err := unifi.NewUnifiDiscovery(context.Background(), nil, cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create discovery service: %w", err)
	}

	logger.Info("Running DiscoverResources...")
	configs, err := svc.DiscoverResources(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	logger.Infof("Discovered %d camera(s)", len(configs))
	for i, cfg := range configs {
		logger.Infof("[%d] Camera: %s", i+1, cfg.Name)
		logger.Infof("    Model: %s", cfg.Model)
		if attrs, ok := cfg.Attributes["rtsp_address"]; ok {
			logger.Infof("    RTSP: %s", attrs)
		}
	}

	return nil
}
