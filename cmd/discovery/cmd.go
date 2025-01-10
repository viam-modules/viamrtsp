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

func realMain() error {
	logger := logging.NewLogger("discovery")

	debug := false
	username := ""
	password := ""
	output := ""

	flag.BoolVar(&debug, "debug", debug, "debug")
	flag.StringVar(&username, "user", username, "username")
	flag.StringVar(&password, "pass", password, "password")
	flag.StringVar(&output, "o", output, "output file")

	flag.Parse()

	if debug {
		logger.SetLevel(logging.DEBUG)
	}

	list, err := viamonvif.DiscoverCameras(username, password, logger, flag.Args())
	if err != nil {
		return err
	}

	for _, l := range list.Cameras {
		logger.Infof("%s %s %s", l.Manufacturer, l.Model, l.SerialNumber)
		for _, u := range l.RTSPURLs {
			logger.Infof("\t%s", u)
		}
	}

	if output != "" {
		j, err := json.Marshal(list.Cameras)
		if err != nil {
			return err
		}

		if err := os.WriteFile(output, j, 0600); err != nil {
			return err
		}
	}

	return nil
}
