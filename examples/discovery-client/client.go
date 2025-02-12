package main

import (
	"context"

	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/robot/client"
	"go.viam.com/rdk/services/discovery"
	"go.viam.com/utils/rpc"
)

func main() {
	logger := logging.NewDebugLogger("client")
	machine, err := client.New(
		context.Background(),
		"<machine-address>", // replace with your machine address, api key etc.
		logger,
		client.WithDialOptions(rpc.WithEntityCredentials(
			"<api-key-id>",
			rpc.Credentials{
				Type:    rpc.CredentialsTypeAPIKey,
				Payload: "<api-key>",
			})),
	)
	if err != nil {
		logger.Fatal(err)
	}

	defer machine.Close(context.Background())

	dis, err := discovery.FromRobot(machine, "<discovery-name>")
	if err != nil {
		logger.Fatal(err)
	}

	extras := map[string]any{}
	extras["User"] = "<onvif-username>" // optional credentials for if your device is ONVIF authenticated
	extras["Pass"] = "<onvif-password>" // can also be configured in the discovery service
	cfgs, err := dis.DiscoverResources(context.Background(), extras)
	if err != nil {
		logger.Fatal(err)
	}
	// print all discovered resources
	for _, cfg := range cfgs {
		logger.Infof("Name: %v\tModel: %v\tAPI: %v", cfg.Name, cfg.Model, cfg.API)
		logger.Infof("Attributes: ", cfg.Attributes)
	}
}
