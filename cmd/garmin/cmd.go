// This package is a standalone binary for trying out Garmin mDNS discovery. It
// browses the local network for Garmin cameras and dumps the result as JSON.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/viam-modules/viamrtsp/garmin"
	"go.viam.com/rdk/logging"
)

func main() {
	if err := realMain(); err != nil {
		log.Fatal(err.Error())
	}
}

func realMain() error {
	debug := flag.Bool("debug", false, "enable debug logging")
	output := flag.String("output", "", "if set, also write the JSON to this file")
	flag.Parse()

	var logger logging.Logger
	if *debug {
		logger = logging.NewDebugLogger("garmin-discovery")
	} else {
		logger = logging.NewLogger("garmin-discovery")
	}

	cameras, err := garmin.DiscoverCameras(context.Background(), logger)
	if err != nil {
		return err
	}

	jsonBytes, err := json.MarshalIndent(cameras, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(jsonBytes))

	if *output != "" {
		//nolint:mnd
		if err := os.WriteFile(*output, jsonBytes, 0o600); err != nil {
			return err
		}
	}

	return nil
}
