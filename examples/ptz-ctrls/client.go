package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"go.viam.com/rdk/components/generic"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/robot/client"
	"go.viam.com/utils/rpc"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
)

func main() {
	logger := logging.NewDebugLogger("client")
	machine, err := client.New(
		context.Background(),
		"<MACHINE_ADDRESS>",
		logger,
		client.WithDialOptions(rpc.WithEntityCredentials(
			/* Replace "<API-KEY-ID>" (including brackets) with your machine's API key ID */
			"<API-KEY-ID>",
			rpc.Credentials{
				Type: rpc.CredentialsTypeAPIKey,
				/* Replace "<API-KEY>" (including brackets) with your machine's API key */
				Payload: "<API-KEY>",
			})),
	)
	if err != nil {
		logger.Fatal(err)
	}

	defer machine.Close(context.Background())
	logger.Info("Resources:")
	logger.Info(machine.ResourceNames())

	ptz, err := generic.FromRobot(machine, "ptz-1")
	if err != nil {
		logger.Error(err)
		return
	}

	// Stop any ongoing pan/tilt/zoom commands
	ptzReturnValue, err := ptz.DoCommand(
		context.Background(),
		map[string]interface{}{
			"command":  "stop",
			"pan_tilt": true,
			"zoom":     true,
		},
	)
	if err != nil {
		logger.Error(err)
		return
	}
	logger.Infof("stopped ongoing ptz controls: %+v", ptzReturnValue)
	time.Sleep(1 * time.Second)

	// Map of keys to release timers
	pressed := make(map[string]*time.Timer)
	var mu sync.Mutex

	// Duration after which we consider the key "released"
	releaseDelay := 100 * time.Millisecond

	fmt.Println("Keyboard Controls:")
	fmt.Println("W/w: Tilt up")
	fmt.Println("S/s: Tilt down")
	fmt.Println("A/a: Pan left")
	fmt.Println("D/d: Pan right")
	fmt.Println("Esc or Ctrl+C: Exit")

	keyboard.Listen(func(key keys.Key) (stop bool, err error) {
		mu.Lock()
		defer mu.Unlock()

		keyStr := key.String() // Convert the key to a string
		fmt.Printf("Key pressed: %s\n", keyStr)

		// If a timer exists for this key, stop and reset it
		if t, ok := pressed[keyStr]; ok {
			t.Stop()
		}

		// Handle pan and tilt commands for WASD keys
		switch keyStr {
		case "W", "w":
			go func() {
				_, err := ptz.DoCommand(context.Background(), map[string]interface{}{
					"command":    "continuous-move",
					"pan_speed":  0.0,
					"tilt_speed": 0.2,
				})
				if err != nil {
					logger.Error(err)
				}
			}()
		case "S", "s":
			go func() {
				_, err := ptz.DoCommand(context.Background(), map[string]interface{}{
					"command":    "continuous-move",
					"pan_speed":  0.0,
					"tilt_speed": -0.2,
				})
				if err != nil {
					logger.Error(err)
				}
			}()
		case "A", "a":
			go func() {
				_, err := ptz.DoCommand(context.Background(), map[string]interface{}{
					"command":    "continuous-move",
					"pan_speed":  -0.2,
					"tilt_speed": 0.0,
				})
				if err != nil {
					logger.Error(err)
				}
			}()
		case "D", "d":
			go func() {
				_, err := ptz.DoCommand(context.Background(), map[string]interface{}{
					"command":    "continuous-move",
					"pan_speed":  0.2,
					"tilt_speed": 0.0,
				})
				if err != nil {
					logger.Error(err)
				}
			}()

		default:
			fmt.Printf("Key pressed: %s\n", keyStr)
		}

		// Start a timer to emulate "key release"
		pressed[keyStr] = time.AfterFunc(releaseDelay, func() {
			mu.Lock()
			defer mu.Unlock()
			delete(pressed, keyStr)

			// Send stop command after key release
			go func() {
				_, err := ptz.DoCommand(context.Background(), map[string]interface{}{
					"command":  "stop",
					"pan_tilt": true,
					"zoom":     false,
				})
				if err != nil {
					logger.Error(err)
				}
				logger.Infof("Key released: %s", keyStr)
			}()
		})

		if key.Code == 27 || key.Code == 3 { // ASCII values for Escape and Ctrl+C
			os.Exit(0)
		}
		return false, nil
	})

}
