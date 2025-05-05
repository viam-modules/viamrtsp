package main

import (
	"context"
	"os"

	"github.com/joho/godotenv"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/robot/client"
	"go.viam.com/rdk/services/discovery"
	"go.viam.com/utils/rpc"
)

func main() {
	logger := logging.NewDebugLogger("client")

	if err := godotenv.Load(); err != nil {
		logger.Warnf("No .env file found: %v", err)
		logger.Info("Make sure to set VIAM_API_KEY, VIAM_API_KEY_ID, VIAM_MACHINE_ADDRESS, and DISCOVERY_NAME environment variables.")
	}

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
	discoveryName, exists := os.LookupEnv("DISCOVERY_NAME")
	if !exists {
		logger.Warn("DISCOVERY_NAME not set")
		os.Exit(1)
	}
	onvifUsername, exists := os.LookupEnv("ONVIF_USERNAME")
	if !exists {
		logger.Warn("ONVIF_USERNAME not set")
	}
	onvifPassword, exists := os.LookupEnv("ONVIF_PASSWORD")
	if !exists {
		logger.Warn("ONVIF_PASSWORD not set")
	}

	machine, err := client.New(
		context.Background(),
		machineAddress, // replace with your machine address, api key etc.
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

	dis, err := discovery.FromRobot(machine, discoveryName)
	if err != nil {
		logger.Fatal(err)
	}

	extras := map[string]any{}
	if onvifUsername != "" && onvifPassword != "" {
		logger.Info("Using ONVIF credentials from environment variables")
		extras["User"] = onvifUsername
		extras["Pass"] = onvifPassword
	}
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
