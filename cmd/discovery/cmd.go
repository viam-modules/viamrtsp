// This package is a binary for trying out onvif discovery
package main

import (
	"context"
	"flag"

	"github.com/erh/viamupnp"
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
	upnpmodel := ""
	upnpmfg := ""
	upnpnetwork := ""
	upnpser := ""


	flag.BoolVar(&debug, "debug", debug, "debug")
	flag.StringVar(&username, "user", username, "username")
	flag.StringVar(&password, "pass", password, "password")
	flag.StringVar(&upnpmodel, "upnpmodel", upnpmodel, "")
	flag.StringVar(&upnpmfg, "upnpmfg", upnpmfg, "")
	flag.StringVar(&upnpser, "upnpmfg", upnpser, "")
	flag.StringVar(&upnpnetwork, "upnpnet", upnpnetwork, "239.255.255.250:1900")

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

	upnpq := viamupnp.DeviceQuery{
		ModelName:    "CAM200",
		Manufacturer: "VisionHitech",
		SerialNumber: "00:26:e6:01:01:51",
		Network:      "239.255.255.250:1900",
	}
	upnp, err := viamupnp.FindHost(context.Background(), logger, upnpq)
	if err != nil {
		logger.Error(err)
	}
	logger.Infof("upnp host %v", upnp)

	upnpq = viamupnp.DeviceQuery{
		ModelName:    "CAM220IP",
		Manufacturer: "VisionHitech",
		SerialNumber: "00:26:E6:10:6A:D9",
		Network:      "239.255.255.250:1900",
	}
	upnp, err = viamupnp.FindHost(context.Background(), logger, upnpq)
	if err != nil {
		logger.Error(err)
	}
	logger.Infof("upnp host %v", upnp)

	upnpq = viamupnp.DeviceQuery{
		ModelName:    "MCF26C_IR",
		Manufacturer: "ONVIF_ICAMERA",
		SerialNumber: "EF00000005083E92",
		Network:      "239.255.255.250:1900",
	}
	upnp, err = viamupnp.FindHost(context.Background(), logger, upnpq)
	if err != nil {
		logger.Error(err)
	}
	logger.Infof("upnp host %v", upnp)

	return nil
}
