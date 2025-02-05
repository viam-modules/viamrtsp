// This package is a binary for trying out onvif discovery
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"maps"
	"net/url"
	"os"
	"slices"
	"time"

	"github.com/viam-modules/viamrtsp/viamonvif"
	"github.com/viam-modules/viamrtsp/viamonvif/device"
	"go.viam.com/rdk/logging"
)

const timeoutDuration = 10 * time.Second

// Config for the disovery command.
type Config struct {
	// the device credentials to use when attempting to authenticate via onvif
	Creds []device.Credentials `json:"creds"`
	// the urls to attempt to connect to as if they were returned from WS-Discovery service as XAddrs
	XAddrs []string `json:"xaddrs"`
}

type options struct {
	config Config
	debug  bool
	output string
}

func main() {
	if err := realMain(); err != nil {
		log.Fatal(err.Error())
	}
}

func realMain() error {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	opts, err := parseOpts()
	if err != nil {
		return err
	}

	var logger logging.Logger
	if opts.debug {
		logger = logging.NewDebugLogger("discovery")
	} else {
		logger = logging.NewLogger("discovery")
	}

	xaddrs := map[string]*url.URL{}
	for _, xaddr := range opts.config.XAddrs {
		u, err := url.Parse(xaddr)
		if err != nil {
			logger.Warnf("invalid config xaddr: %s", xaddr)
			continue
		}
		xaddrs[u.Host] = u
	}

	urls := slices.Collect(maps.Values(xaddrs))
	list, err := viamonvif.DiscoverCameras(timeoutCtx, opts.config.Creds, urls, logger)
	if err != nil {
		return err
	}

	if opts.output != "" {
		j, err := json.Marshal(list.Cameras)
		if err != nil {
			return err
		}

		//nolint:mnd
		if err := os.WriteFile(opts.output, j, 0o600); err != nil {
			return err
		}
	}

	return nil
}

func parseOpts() (options, error) {
	debug := false
	genConfig := false
	configFile := "./config.json"
	output := "./output.json"
	var zero options

	flag.BoolVar(&debug, "debug", debug, "debug")
	flag.BoolVar(&genConfig, "gen_config", genConfig, "generate config file template")
	flag.StringVar(&configFile, "config", configFile, "path to json config file.")
	flag.StringVar(&output, "output", output, "output file")
	flag.Parse()

	if genConfig {
		b, err := json.Marshal(Config{XAddrs: []string{"192.168.1.1"}, Creds: []device.Credentials{{User: "username", Pass: "password"}}})
		if err != nil {
			return zero, err
		}
		if _, err := os.Stat(configFile); err == nil {
			return zero, fmt.Errorf("can't create config file template as %s file or directory already exists", configFile)
		}

		//nolint:mnd
		if err := os.WriteFile(configFile, b, 0o600); err != nil {
			return zero, err
		}
		os.Exit(0)
	}

	configBytes, err := os.ReadFile(configFile)
	var config Config
	if err != nil {
		return zero, err
	}
	if err := json.Unmarshal(configBytes, &config); err != nil {
		return zero, err
	}

	return options{
		debug:  debug,
		output: output,
		config: config,
	}, nil
}
