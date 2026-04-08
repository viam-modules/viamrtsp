package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
	ptzpb "go.viam.com/api/component/ptz/v1"
	"go.viam.com/rdk/components/ptz"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/robot/client"
	"go.viam.com/utils/rpc"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
)

func main() {
	logger := logging.NewDebugLogger("ptz-api-client")

	if err := godotenv.Load(); err != nil {
		logger.Errorf("No .env file found: %v", err)
		logger.Info("Make sure to set VIAM_API_KEY, VIAM_API_KEY_ID, and VIAM_MACHINE_ADDRESS environment variables.")
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

	resourceName := "ptz-1"
	if len(os.Args) >= 2 {
		resourceName = os.Args[1]
	}

	ptzClient, err := ptz.FromProvider(machine, resourceName)
	if err != nil {
		logger.Error(err)
		return
	}

	// Stop any ongoing pan/tilt/zoom commands
	if err := ptzClient.Stop(context.Background(), boolPtr(true), boolPtr(true), nil); err != nil {
		logger.Error(err)
		return
	}
	logger.Info("Stopped ongoing PTZ controls")
	time.Sleep(1 * time.Second)

	pressed := make(map[string]*time.Timer)
	var timerMu sync.Mutex
	var cmdMu sync.Mutex
	releaseDelay := 10 * time.Millisecond

	fmt.Println("Keyboard Controls (PTZ API):")
	fmt.Println("W/w: Tilt up")
	fmt.Println("S/s: Tilt down")
	fmt.Println("A/a: Pan left")
	fmt.Println("D/d: Pan right")
	fmt.Println("R/r: Zoom in")
	fmt.Println("F/f: Zoom out")
	fmt.Println("Esc or Ctrl+C: Exit")

	sendContinuous := func(pan, tilt, zoom float64) {
		cmdMu.Lock()
		defer cmdMu.Unlock()
		err := ptzClient.Move(context.Background(), &ptz.MoveCommand{
			Continuous: &ptzpb.ContinuousMove{
				Velocity: &ptzpb.Velocity{
					Pan:  pan,
					Tilt: tilt,
					Zoom: zoom,
				},
			},
		}, nil)
		if err != nil {
			logger.Error(err)
		}
	}

	sendStop := func() {
		cmdMu.Lock()
		defer cmdMu.Unlock()
		if err := ptzClient.Stop(context.Background(), boolPtr(true), boolPtr(true), nil); err != nil {
			logger.Error(err)
		}
	}

	keyboard.Listen(func(key keys.Key) (stop bool, err error) {
		timerMu.Lock()
		defer timerMu.Unlock()

		keyStr := key.String()
		fmt.Printf("Key pressed: %s\n", keyStr)

		if t, ok := pressed[keyStr]; ok {
			t.Stop()
		}

		switch keyStr {
		case "W", "w":
			go sendContinuous(0.0, 0.5, 0.0)
		case "S", "s":
			go sendContinuous(0.0, -0.5, 0.0)
		case "A", "a":
			go sendContinuous(-0.5, 0.0, 0.0)
		case "D", "d":
			go sendContinuous(0.5, 0.0, 0.0)
		case "R", "r":
			go sendContinuous(0.0, 0.0, 0.5)
		case "F", "f":
			go sendContinuous(0.0, 0.0, -0.5)
		default:
			fmt.Printf("Key pressed: %s\n", keyStr)
		}

		pressed[keyStr] = time.AfterFunc(releaseDelay, func() {
			timerMu.Lock()
			defer timerMu.Unlock()
			delete(pressed, keyStr)
			go sendStop()
			logger.Infof("Key released: %s", keyStr)
		})

		if key.Code == 27 || key.Code == 3 {
			logger.Info("Exiting...")
			sendStop()
			logger.Info("Stopped all PTZ controls.")
			os.Exit(0)
		}
		return false, nil
	})
}

func boolPtr(v bool) *bool {
	return &v
}
