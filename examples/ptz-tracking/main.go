// Package main is a test script that continuously points a PTZ camera at the
// end effector of a robot arm using the Viam frame system.
//
// Setup:
//   - universal-arm: UR7e arm, parented to world
//   - ptz arm: onvif-ptz arm, parented to world at a fixed offset
//   - motion-1: motion service (rdk:builtin:builtin)
//
// The script uses motionSvc.GetPose with ptzName+"_origin" as the destination
// frame to get the arm EE directly in the PTZ's mount frame. The frame system
// handles all the math from the Viam config — no hardcoded constants needed.
//
// The "_origin" frame is the PTZ's static base/mount frame. Using it (rather
// than ptzName directly) avoids a feedback loop: ptzName would require querying
// the PTZ end-effector via FK → ONVIF GetStatus on every iteration.
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

	"github.com/joho/godotenv"
	"go.viam.com/rdk/components/arm"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/robot/client"
	"go.viam.com/rdk/services/motion"
	"go.viam.com/utils/rpc"
)

const (
	armName       = "universal-arm"
	ptzName       = "ptz-client-LOREXSystems-LNZ43P4A-ND012501000714-url2"
	motionSvcName = "motion-1"
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

	motionSvc, err := motion.FromProvider(machine, motionSvcName)
	if err != nil {
		logger.Fatalf("Motion service not found: %v", err)
	}

	logger.Infof("Starting tracking loop: pointing %s at end effector of %s", ptzName, armName)

	for {
		if ctx.Err() != nil {
			logger.Info("Shutting down")
			return
		}

		// Get arm EE expressed in the PTZ's mount frame.
		// ptzName+"_origin" is the PTZ's static base frame — using it avoids
		// querying PTZ joint positions (ONVIF GetStatus) on every iteration.
		// The frame system reads translation+orientation from the Viam config.
		target, err := motionSvc.GetPose(ctx, armName, ptzName+"_origin", nil, nil)
		if err != nil {
			logger.Warnf("GetPose failed: %v", err)
			continue
		}

		pt := target.Pose().Point()
		panDeg := math.Atan2(pt.Y, pt.X) * 180 / math.Pi
		tiltDeg := math.Atan2(-pt.Z, math.Sqrt(pt.X*pt.X+pt.Y*pt.Y)) * 180 / math.Pi
		logger.Infof("EE in PTZ local (mm): x=%.1f y=%.1f z=%.1f → pan=%.1f° tilt=%.1f°", pt.X, pt.Y, pt.Z, panDeg, tiltDeg)

		// MoveToPosition runs analytical IK (atan2) and sends one ONVIF AbsoluteMove.
		// Blocks until camera finishes moving — naturally rate-limits the loop.
		if err := ptzArm.MoveToPosition(ctx, target.Pose(), nil); err != nil {
			logger.Warnf("MoveToPosition failed: %v", err)
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
