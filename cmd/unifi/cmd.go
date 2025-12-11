// Package main is a binary for testing Unifi NVR camera discovery
package main

import (
	"context"
	"flag"
	"fmt"

	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"

	"github.com/viam-modules/viamrtsp/unifi"
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
		return fmt.Errorf("nvr IP address is required (-nvr)")
	}
	if token == "" {
		return fmt.Errorf("unifi API token is required (-token)")
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
		fmt.Printf("\n[%d] Camera: %s\n", i+1, cfg.Name)
		fmt.Printf("    Model: %s\n", cfg.Model)
		if attrs, ok := cfg.Attributes["rtsp_address"]; ok {
			fmt.Printf("    RTSP: %s\n", attrs)
		}
	}

	return nil
}
