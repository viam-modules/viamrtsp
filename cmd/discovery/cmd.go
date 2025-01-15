// This package is a binary for trying out onvif discovery
package main

import (
	"encoding/json"
	"flag"
	"maps"
	"net/url"
	"os"
	"slices"

	"github.com/viam-modules/viamrtsp/viamonvif"
	"go.viam.com/rdk/logging"
)

func main() {
	err := realMain()
	if err != nil {
		panic(err)
	}
}

func ParseOpts() (Options, error) {
	debug := false
	configFile := "config.json"
	output := ""
	var zero Options

	flag.BoolVar(&debug, "debug", debug, "debug")
	flag.StringVar(&configFile, "c", configFile, "path to json config file with structure [{'user': '...', 'pass': '...'}]")
	flag.StringVar(&output, "o", output, "output file")
	flag.Parse()

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
	Creds  []viamonvif.Credentials `json:"creds"`
	XAddrs []string                `json:"xaddrs"`
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

	for _, l := range list.Cameras {
		logger.Infof("%s %s %s", l.Manufacturer, l.Model, l.SerialNumber)
		for _, u := range l.RTSPURLs {
			logger.Infof("\t%s", u)
		}
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
