package main

import (
	"context"
	"os"
	"time"

	"github.com/joho/godotenv"
	vsapi "github.com/viam-modules/viamrtsp/videostore/src/videostore_api_go"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/robot/client"
	"go.viam.com/utils/rpc"
)

func main() {
	logger := logging.NewDebugLogger("client")

	if err := godotenv.Load(); err != nil {
		logger.Errorf("No .env file found: %v", err)
		logger.Info("Make sure to set VIAM_API_KEY, VIAM_API_KEY_ID, and VIAM_MACHINE_ADDRESS environment variables.")
	}
	// Get credentials from environment variables with fallbacks
	apiKeyID, exists := os.LookupEnv("VIAM_API_KEY_ID")
	if !exists {
		logger.Error("VIAM_API_KEY_ID not set")
		os.Exit(1)
	}
	apiKey, exists := os.LookupEnv("VIAM_API_KEY")
	if !exists {
		logger.Error("VIAM_API_KEY not set")
		os.Exit(1)
	}
	machineAddress, exists := os.LookupEnv("VIAM_MACHINE_ADDRESS")
	if !exists {
		logger.Error("VIAM_MACHINE_ADDRESS not set")
		os.Exit(1)
	}

	machine, err := client.New(
		context.Background(),
		machineAddress,
		logger,
		client.WithDialOptions(rpc.WithEntityCredentials(
			apiKeyID,
			rpc.Credentials{
				Type:    rpc.CredentialsTypeAPIKey,
				Payload: apiKey,
			})),
	)
	if err != nil {
		logger.Fatal(err)
	}
	defer machine.Close(context.Background())

	resourceName := "vs-1"
	videoStore, err := vsapi.FromRobot(machine, resourceName)
	if err != nil {
		logger.Errorf("Failed to get video store resource: %v", err)
		os.Exit(1)
	}
	now := time.Now()
	from := now.Add(-60 * time.Second).Format("2006-01-02_15-04-05")
	to := now.Add(-30 * time.Second).Format("2006-01-02_15-04-05")

	ctx := context.Background()
	// video, err := videoStore.Fetch(ctx, from, to)
	// if err != nil {
	// 	logger.Errorf("Failed to save video segment: %v", err)
	// 	os.Exit(1)
	// }
	// logger.Infof("Fetched video segment of length %d bytes", len(video))
	ioWriter := os.Stdout
	err = videoStore.FetchStream(ctx, from, to, ioWriter)
	if err != nil {
		logger.Errorf("Failed to fetch video segment stream: %v", err)
		os.Exit(1)
	}
}
