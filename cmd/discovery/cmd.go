package main

import (
	"flag"
	"fmt"

	"go.viam.com/rdk/logging"

	"github.com/viam-modules/viamrtsp/viamonvif"
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
		fmt.Printf("%s %s %s\n", l.Manufacturer, l.Model, l.SerialNumber)
		for _, u := range l.RTSPURLs {
			fmt.Printf("\t%s\n", u)
		}
	}

	return nil

}
