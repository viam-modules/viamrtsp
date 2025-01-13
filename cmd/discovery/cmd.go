// This package is a binary for trying out onvif discovery
package main

import (
	"encoding/json"
	"flag"
	"os"

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

	config, err := os.ReadFile(configFile)
	var creds []viamonvif.Credentials
	if err != nil {
		return zero, err
	}
	if err := json.Unmarshal(config, &creds); err != nil {
		return zero, err
	}

	return Options{
		Debug:  debug,
		Output: output,
		Creds:  creds,
	}, nil
}

type Options struct {
	Creds  []viamonvif.Credentials
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

	list, err := viamonvif.DiscoverCameras(opts.Creds, logger)
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
