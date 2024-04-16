package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/robot/client"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger := logging.NewDebugLogger("client")

	robot, err := client.New(
		ctx,
		"localhost:8080",
		logger,
	)
	if err != nil {
		logger.Fatal(err)
	}

	defer robot.Close(ctx)

	logger.Info("Resources:")
	logger.Info(robot.ResourceNames())

	ipCam, err := camera.FromRobot(robot, "ip-cam")
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			err = fmt.Errorf("%v: this likely means viam-server could not register/start the resource properly; check logs to verify", err)
		}
		logger.Fatal(err)
	}
	stream, err := ipCam.Stream(ctx)
	if err != nil {
		logger.Fatal(err)
	}
	_, _, err = stream.Next(ctx)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Info("All tests passed! Success :)")
}
