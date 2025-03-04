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
	model := ""
	mfg := ""
	network := ""
	serial := ""

	// flag.BoolVar(&debug, "debug", debug, "debug")
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

	upnp, hostmap, err := viamupnp.FindHost(context.Background(), logger, []viamupnp.DeviceQuery{upnpq}, true)
	if err != nil {
		logger.Error(err)
	}
	logger.Infof("upnp host %s", upnp)
	for _, host := range upnp {
		logger.Infof("host: %v", host)
		logger.Infof("query: %v", hostmap[host])
	}

	return nil
}
