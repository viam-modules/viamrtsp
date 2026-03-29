// Package main is a test script that continuously points a PTZ camera at the
// end effector of a UR5 arm using the Viam frame system.
//
// Setup:
//   - arm-1: fake UR5e arm, parented to world
//   - ptz arm: onvif-ptz arm, parented to world at a fixed offset
//   - motion-1: motion service (rdk:builtin:builtin)
//
// The motion service GetPose call does the full frame chain in one shot:
// arm-1 EE → world → PTZ local frame. The result is passed directly to
// PTZ.MoveToPosition which runs analytical IK (atan2).
//
// Usage:
//
//	cp .env.example .env  # fill in credentials
//	go run main.go
package main

import (
	"context"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"go.viam.com/rdk/components/arm"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/referenceframe"
	"go.viam.com/rdk/robot/client"
	"go.viam.com/rdk/services/motion"
	"go.viam.com/utils/rpc"
)

const (
	armName       = "arm-1"
	ptzName       = "ptz-client-LOREXSystems-LNZ43P4A-ND012501000714-url2"
	motionSvcName = "motion-1"
	pollInterval  = 200 * time.Millisecond

	// PTZ world position from frame config (mm). Must match the frame block in Viam config.
	ptzWorldX = 100.0
	ptzWorldY = 100.0
	ptzWorldZ = 0.0
)

func main() {
	logger := logging.NewDebugLogger("ptz-tracking")

	if err := godotenv.Load(); err != nil {
		logger.Warnf("No .env file found: %v", err)
		logger.Info("Set VIAM_API_KEY, VIAM_API_KEY_ID, and VIAM_MACHINE_ADDRESS")
	}

	apiKeyID := mustEnv("VIAM_API_KEY_ID", logger)
	apiKey := mustEnv("VIAM_API_KEY", logger)
	machineAddress := mustEnv("VIAM_MACHINE_ADDRESS", logger)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	machine, err := client.New(
		ctx,
		machineAddress,
		logger,
		client.WithDialOptions(rpc.WithEntityCredentials(
			apiKeyID,
			rpc.Credentials{
				Type:    rpc.CredentialsTypeAPIKey,
				Payload: apiKey,
			},
		)),
	)
	if err != nil {
		logger.Fatal(err)
	}
	defer machine.Close(context.Background())

	ptzArm, err := arm.FromRobot(machine, ptzName)
	if err != nil {
		logger.Fatalf("PTZ arm not found: %v — available: %v", err, machine.ResourceNames())
	}

	// motionSvc, err := motion.FromRobot(machine, motionSvcName)
	motionSvc, err := motion.FromProvider(machine, motionSvcName)
	if err != nil {
		logger.Fatalf("Motion service not found: %v", err)
	}

	logger.Infof("Starting tracking loop: pointing %s at end effector of %s", ptzName, armName)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var lastPan, lastTilt float64
	const deadbandRad = 0.02 // ~1 degree

	for {
		select {
		case <-ctx.Done():
			logger.Info("Shutting down")
			return
		case <-ticker.C:
			// Get arm EE in world frame — does not query the PTZ, avoids feedback loop.
			eeInWorld, err := motionSvc.GetPose(ctx, armName, "world", nil, nil)
			if err != nil {
				logger.Warnf("GetPose failed: %v", err)
				continue
			}
			pt := eeInWorld.Pose().Point()

			// Vector from PTZ to arm EE in world frame.
			// PTZ orientation is th=0 so world frame == PTZ local frame (no rotation needed).
			x := pt.X - ptzWorldX
			y := pt.Y - ptzWorldY
			z := pt.Z - ptzWorldZ
			logger.Infof("EE relative to PTZ (mm): (%.1f, %.1f, %.1f)", x, y, z)

			// Analytical IK matching MoveToPosition in arm.go
			pan := math.Atan2(y, x)
			tilt := math.Atan2(-z, math.Sqrt(x*x+y*y))
			logger.Infof("target pan=%.2f° tilt=%.2f°", pan*180/math.Pi, tilt*180/math.Pi)

			if math.Abs(pan-lastPan) < deadbandRad && math.Abs(tilt-lastTilt) < deadbandRad {
				logger.Infof("within deadband, skipping")
				continue
			}
			lastPan, lastTilt = pan, tilt

			// MoveToJointPositions bypasses motion planning — sends ONVIF AbsoluteMove directly.
			if err := ptzArm.MoveToJointPositions(ctx, []referenceframe.Input{pan, tilt}, nil); err != nil {
				logger.Warnf("MoveToJointPositions failed: %v", err)
			}
		}
	}
}

func mustEnv(key string, logger logging.Logger) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		logger.Fatalf("%s not set", key)
	}
	return val
}
