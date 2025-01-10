// This package is a binary for trying out onvif discovery
package main

import (
	"flag"

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

	flag.BoolVar(&debug, "debug", debug, "debug")
	flag.StringVar(&username, "user", username, "username")
	flag.StringVar(&password, "pass", password, "password")

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
	return nil
}
