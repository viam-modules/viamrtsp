package main

import (
	"context"

	"github.com/viam-modules/viamrtsp"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/robot/client"
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

	qs := []resource.DiscoveryQuery{
		{
			API:   camera.API,
			Model: viamrtsp.ModelAgnostic,
			Extra: map[string]interface{}{
				"username": "<onvif-username>", // optional credentials for if your device is ONVIF authenticated
				"password": "<onvif-password>",
			},
		},
	}
	discoveries, err := machine.DiscoverComponents(context.Background(), qs)
	for _, discovery := range discoveries {
		logger.Infof("Discovered: %v", discovery.Results)
	}
	if err != nil {
		logger.Fatalf("Failed to discover due to: %v", err)
	}
}
