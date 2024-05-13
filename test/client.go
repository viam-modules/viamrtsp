// This package is a test client for RTSP cam integration tests
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/robot/client"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Fatal error: %v", err)
	}
	log.Println("All tests passed! Success :)")
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger := logging.NewDebugLogger("client")

	robot, err := client.New(
		ctx,
		"localhost:8080",
		logger,
	)
	if err != nil {
		return err
	}
	defer func() {
		if err := robot.Close(ctx); err != nil {
			logger.Errorf("failed to close robot client: %v", err)
		}
	}()

	logger.Info("Resources:")
	logger.Info(robot.ResourceNames())

	ipCam, err := camera.FromRobot(robot, "ip-cam")
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("%w: this likely means viam-server could not register/start the resource properly; check logs to verify", err)
		}
		return err
	}
	stream, err := ipCam.Stream(ctx)
	if err != nil {
		return err
	}
	_, _, err = stream.Next(ctx)
	if err != nil {
		return err
	}

	return nil
}
