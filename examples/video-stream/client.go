package main

import (
	"context"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/robot/client"
	"go.viam.com/rdk/services/video"
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

	ctx := context.Background()
	machine, err := client.New(
		ctx,
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
	defer machine.Close(ctx)

	// Get the video service (your videostore-backed implementation).
	// _, err = video.FromRobot(machine, resourceName)
	vs, err := video.FromProvider(machine, resourceName)
	if err != nil {
		logger.Fatalf("failed to get video service %q: %v", resourceName, err)
	}

	// // Example: fetch last 30 seconds of video.
	end := time.Now().UTC()
	start := end.Add(-30 * time.Second)

	logger.Infof("Calling GetVideo from %s to %s", start.Format(time.RFC3339), end.Format(time.RFC3339))

	// videoCodec / videoContainer are hints; your service currently ignores codec
	// and uses container (optionally) through extra["container"] or videoContainer.
	videoCodec := ""     // let server decide
	videoContainer := "" // or "mp4"/"fmp4" if you want to force it

	ch, err := vs.GetVideo(ctx, start, end, videoCodec, videoContainer, nil)
	if err != nil {
		logger.Fatalf("GetVideo failed: %v", err)
	}

	var totalBytes int
	chunkIdx := 0
	for chunk := range ch {
		if chunk == nil {
			continue
		}
		n := len(chunk.Data)
		totalBytes += n
		chunkIdx++
		logger.Infof("received chunk %d: %d bytes", chunkIdx, n)
	}

	logger.Infof("stream complete: %d chunks, %d total bytes", chunkIdx, totalBytes)
}
