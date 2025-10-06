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

	resourceName := "vs-1"
	if len(os.Args) > 1 && os.Args[1] != "" {
		resourceName = os.Args[1]
		logger.Infof("Using VideoStore resource: %s", resourceName)
	} else {
		logger.Infof("No resource arg provided; defaulting to %s", resourceName)
	}

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

	videoStore, err := vsapi.FromRobot(machine, resourceName)
	if err != nil {
		logger.Errorf("Failed to get video store resource: %v", err)
		os.Exit(1)
	}
	now := time.Now()
	from := now.Add(-60 * time.Second).Format("2006-01-02_15-04-05")
	to := now.Add(-50 * time.Second).Format("2006-01-02_15-04-05")

	ctx := context.Background()

	// Test that video bytes can be streamed and written to a file
	ioWriter := os.Stdout
	err = videoStore.FetchStream(ctx, from, to, ioWriter)
	if err != nil {
		logger.Errorf("Failed to fetch video segment stream: %v", err)
		os.Exit(1)
	}
}
