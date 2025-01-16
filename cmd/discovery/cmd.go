// This package is a binary for trying out onvif discovery
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"maps"
	"net/url"
	"os"
	"slices"

	"github.com/viam-modules/viamrtsp/viamonvif"
	"github.com/viam-modules/viamrtsp/viamonvif/device"
	"go.viam.com/rdk/logging"
)

func main() {
	err := realMain()
	if err != nil {
		log.Print(err.Error())
		os.Exit(1)
	}
}

func ParseOpts() (Options, error) {
	debug := false
	genConfig := false
	configFile := "./config.json"
	output := "./output.json"
	var zero Options

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

	return Options{
		Debug:  debug,
		Output: output,
		Config: config,
	}, nil
}

type Config struct {
	Creds  []device.Credentials `json:"creds"`
	XAddrs []string             `json:"xaddrs"`
}

type Options struct {
	Config Config
	Debug  bool
	Output string
}

func realMain() error {
	opts, err := ParseOpts()
	if err != nil {
		return err
	}

	var logger logging.Logger
	if opts.Debug {
		logger = logging.NewDebugLogger("discovery")
	} else {
		logger = logging.NewLogger("discovery")
	}

	xaddrs := map[string]*url.URL{}
	for _, xaddr := range opts.Config.XAddrs {
		u, err := url.Parse(xaddr)
		if err != nil {
			logger.Warnf("invalid config xaddr: %s", xaddr)
			continue
		}
		xaddrs[u.Host] = u
	}

	urls := slices.Collect(maps.Values(xaddrs))
	list, err := viamonvif.DiscoverCameras(opts.Config.Creds, urls, logger)
	if err != nil {
		return err
	}

	if opts.Output != "" {
		j, err := json.Marshal(list.Cameras)
		if err != nil {
			return err
		}

		//nolint:mnd
		if err := os.WriteFile(opts.Output, j, 0o600); err != nil {
			return err
		}
	}

	return nil
}
