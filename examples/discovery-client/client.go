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
		"rawr-seanorg-main.nahz2tk7xm.viam.cloud",
		logger,
		client.WithDialOptions(rpc.WithEntityCredentials( 
			"23c7c65f-7a77-401d-b551-b9288ec73b16",
			rpc.Credentials{
				Type:    rpc.CredentialsTypeAPIKey, 
				Payload: "0tgr1rhxjltjpwxnvcwfalnn0ecpxr9y",
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
