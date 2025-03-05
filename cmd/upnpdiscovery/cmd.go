// This package is a binary for trying out onvif discovery
package main

import (
	"context"
	"flag"

	"github.com/erh/viamupnp"
	"go.viam.com/rdk/logging"
)

func main() {
	err := realMain()
	if err != nil {
		panic(err)
	}
}

func realMain() error {
	logger := logging.NewLogger("upnp-discovery")

	debug := true
	// rootOnly := false will search all services on a network, rather than just the root services.
	// setting this to true is useful when you already know what services/endpoints you wish to use from a device.
	rootOnly := false
	model := ""
	mfg := ""
	network := ""
	serial := ""

	flag.BoolVar(&debug, "debug", debug, "debug")
	flag.BoolVar(&rootOnly, "rootOnly", rootOnly, "run in root only mode")
	flag.StringVar(&model, "model", model, "model of device")
	flag.StringVar(&mfg, "make", mfg, "make of device")
	flag.StringVar(&serial, "serial", serial, "serial number of device")
	flag.StringVar(&network, "network", network, "network to cast to")

	flag.Parse()

	if debug {
		logger.SetLevel(logging.DEBUG)
	}

	upnpq := viamupnp.DeviceQuery{
		ModelName:    model,
		Manufacturer: mfg,
		Network:      network,
		SerialNumber: serial,
	}

	logger.Infof("running upnp query with arguments%#v", upnpq)

	upnp, hostmap, err := viamupnp.FindHost(context.Background(), logger, []viamupnp.DeviceQuery{upnpq}, rootOnly)
	if err != nil {
		logger.Error(err)
		return err
	}
	logger.Infof("upnp host %s", upnp)
	for _, host := range upnp {
		logger.Infof("host: %v", host)
		logger.Infof("query: %v", hostmap[host])
	}

	return nil
}
