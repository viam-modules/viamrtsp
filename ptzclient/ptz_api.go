package ptzclient

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/viam-modules/viamrtsp"
	onvifptz "github.com/viam-modules/viamrtsp/viamonvif/ptz"
	ptzpb "go.viam.com/api/component/ptz/v1"
	rdkptz "go.viam.com/rdk/components/ptz"
	"go.viam.com/rdk/resource"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// PTZModel is the model for the ONVIF PTZ client using the PTZ API.
var PTZModel = viamrtsp.Family.WithModel("onvif-ptz")

func init() {
	resource.RegisterComponent(
		rdkptz.API,
		PTZModel,
		resource.Registration[resource.Resource, *Config]{
			Constructor: newOnvifPtzClientClient,
		},
	)
}

// GetStatus returns the current PTZ position and movement status.
func (s *onvifPtzClient) GetStatus(_ context.Context, _ map[string]interface{}) (*rdkptz.Status, error) {
	profileToken, err := s.profileToken()
	if err != nil {
		return nil, err
	}

	var envelope CustomGetStatusEnvelope
	if _, err := s.callPTZMethod(onvifptz.GetStatus{ProfileToken: profileToken}, &envelope); err != nil {
		return nil, err
	}

	status := envelope.Body.GetResponse.PTZStatus
	return &rdkptz.Status{
		Position: &ptzpb.Pose{
			PanTilt: &ptzpb.Vector2D{
				X:     status.Position.PanTilt.X,
				Y:     status.Position.PanTilt.Y,
				Space: status.Position.PanTilt.Space,
			},
			Zoom: &ptzpb.Vector1D{
				X:     status.Position.Zoom.X,
				Space: status.Position.Zoom.Space,
			},
		},
		PanTiltStatus: mapMoveStatus(status.MoveStatus.PanTilt),
		ZoomStatus:    mapMoveStatus(status.MoveStatus.Zoom),
		UtcTime:       parseOnvifTime(status.UtcTime),
	}, nil
}

// GetCapabilities returns standardized PTZ capabilities.
func (s *onvifPtzClient) GetCapabilities(_ context.Context, _ map[string]interface{}) (*rdkptz.Capabilities, error) {
	var caps []*ptzpb.PTZMoveCapability
	for _, key := range []string{"absolute", "relative", "continuous"} {
		mvmt, ok := s.cfg.Movements[key]
		if !ok {
			continue
		}
		caps = append(caps, movementToCapability(key, mvmt))
	}
	supportsStatus := true
	supportsStop := true
	return &rdkptz.Capabilities{
		MoveCapabilities: caps,
		SupportsStatus:   &supportsStatus,
		SupportsStop:     &supportsStop,
	}, nil
}

// Stop halts any ongoing PTZ movement. If panTilt or zoom are nil, the device default is used.
func (s *onvifPtzClient) Stop(_ context.Context, panTilt, zoom *bool, _ map[string]interface{}) error {
	cmd := map[string]interface{}{"command": "stop"}
	if panTilt != nil {
		cmd["pan_tilt"] = *panTilt
	}
	if zoom != nil {
		cmd["zoom"] = *zoom
	}
	_, err := s.handleStop(cmd)
	return err
}

// Move executes a PTZ movement command (continuous, relative, or absolute).
func (s *onvifPtzClient) Move(_ context.Context, cmd *rdkptz.MoveCommand, _ map[string]interface{}) error {
	if cmd == nil {
		return errors.New("move command is required")
	}
	if countMoveTypes(cmd) != 1 {
		return errors.New("move command must include exactly one of continuous, relative, or absolute")
	}

	switch {
	case cmd.Continuous != nil:
		return s.moveContinuous(cmd.Continuous)
	case cmd.Relative != nil:
		return s.moveRelative(cmd.Relative)
	case cmd.Absolute != nil:
		return s.moveAbsolute(cmd.Absolute)
	default:
		return errors.New("move command must include a continuous, relative, or absolute request")
	}
}

func (s *onvifPtzClient) moveContinuous(req *ptzpb.ContinuousMove) error {
	cmd := map[string]interface{}{"command": "continuous-move"}
	if req.Velocity != nil {
		cmd["pan_speed"] = req.Velocity.Pan
		cmd["tilt_speed"] = req.Velocity.Tilt
		cmd["zoom_speed"] = req.Velocity.Zoom
	}
	if req.Timeout != nil {
		timeout, err := durationToXSD(req.Timeout.AsDuration())
		if err != nil {
			return err
		}
		cmd["timeout"] = timeout
	}
	_, err := s.handleContinuousMove(cmd)
	return err
}

func (s *onvifPtzClient) moveRelative(req *ptzpb.RelativeMove) error {
	cmd := map[string]interface{}{"command": "relative-move"}
	if req.Translation != nil {
		if req.Translation.PanTilt != nil {
			cmd["pan"] = req.Translation.PanTilt.X
			cmd["tilt"] = req.Translation.PanTilt.Y
			if req.Translation.PanTilt.Space != "" {
				cmd["pan_tilt_space"] = req.Translation.PanTilt.Space
			}
		}
		if req.Translation.Zoom != nil {
			cmd["zoom_translation"] = req.Translation.Zoom.X
		}
	}
	if req.Speed != nil {
		cmd["pan_speed"] = req.Speed.Pan
		cmd["tilt_speed"] = req.Speed.Tilt
		cmd["zoom_speed"] = req.Speed.Zoom
	}
	_, err := s.handleRelativeMove(cmd)
	return err
}

func (s *onvifPtzClient) moveAbsolute(req *ptzpb.AbsoluteMove) error {
	cmd := map[string]interface{}{"command": "absolute-move"}
	if req.Position == nil || req.Position.PanTilt == nil || req.Position.Zoom == nil {
		return errors.New("absolute move requires position pan_tilt and zoom")
	}
	cmd["pan_position"] = req.Position.PanTilt.X
	cmd["tilt_position"] = req.Position.PanTilt.Y
	cmd["zoom_position"] = req.Position.Zoom.X
	if req.Speed != nil {
		cmd["pan_speed"] = req.Speed.Pan
		cmd["tilt_speed"] = req.Speed.Tilt
		cmd["zoom_speed"] = req.Speed.Zoom
	}
	_, err := s.handleAbsoluteMove(cmd)
	return err
}

func countMoveTypes(cmd *rdkptz.MoveCommand) int {
	count := 0
	if cmd.Continuous != nil {
		count++
	}
	if cmd.Relative != nil {
		count++
	}
	if cmd.Absolute != nil {
		count++
	}
	return count
}

func movementToCapability(kind string, mvmt PTZMovement) *ptzpb.PTZMoveCapability {
	c := &ptzpb.PTZMoveCapability{
		Type: moveTypeFromString(kind),
	}
	if r := range2DFromSpace(mvmt.PanTilt); r != nil {
		c.PanTilt = r
	}
	if r := range1DFromSpace(mvmt.Zoom); r != nil {
		c.Zoom = r
	}
	return c
}

func moveTypeFromString(kind string) ptzpb.PTZMoveType {
	switch strings.ToLower(kind) {
	case "absolute":
		return ptzpb.PTZMoveType_PTZ_MOVE_TYPE_ABSOLUTE
	case "relative":
		return ptzpb.PTZMoveType_PTZ_MOVE_TYPE_RELATIVE
	case "continuous":
		return ptzpb.PTZMoveType_PTZ_MOVE_TYPE_CONTINUOUS
	default:
		return ptzpb.PTZMoveType_PTZ_MOVE_TYPE_UNSPECIFIED
	}
}

func range2DFromSpace(space PanTiltSpace) *ptzpb.Range2D {
	if space.Space == "" && space.XMin == 0 && space.XMax == 0 && space.YMin == 0 && space.YMax == 0 {
		return nil
	}
	return &ptzpb.Range2D{
		XMin:  space.XMin,
		XMax:  space.XMax,
		YMin:  space.YMin,
		YMax:  space.YMax,
		Space: space.Space,
	}
}

func range1DFromSpace(space ZoomSpace) *ptzpb.Range1D {
	if space.Space == "" && space.XMin == 0 && space.XMax == 0 {
		return nil
	}
	return &ptzpb.Range1D{
		Min:   space.XMin,
		Max:   space.XMax,
		Space: space.Space,
	}
}

func mapMoveStatus(status string) ptzpb.PTZMoveStatus {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "IDLE":
		return ptzpb.PTZMoveStatus_PTZ_MOVE_STATUS_IDLE
	case "MOVING":
		return ptzpb.PTZMoveStatus_PTZ_MOVE_STATUS_MOVING
	default:
		return ptzpb.PTZMoveStatus_PTZ_MOVE_STATUS_UNKNOWN
	}
}

func parseOnvifTime(raw string) *timestamppb.Timestamp {
	if raw == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return timestamppb.New(t)
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return timestamppb.New(t)
	}
	return nil
}

func durationToXSD(d time.Duration) (string, error) {
	if d < 0 {
		return "", errors.New("timeout must be non-negative")
	}
	totalSeconds := int64(d / time.Second)
	nanos := int64(d % time.Second)
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	var sb strings.Builder
	sb.WriteString("PT")
	if hours > 0 {
		sb.WriteString(fmt.Sprintf("%dH", hours))
	}
	if minutes > 0 {
		sb.WriteString(fmt.Sprintf("%dM", minutes))
	}
	if seconds > 0 || nanos > 0 || (hours == 0 && minutes == 0) {
		if nanos == 0 {
			sb.WriteString(fmt.Sprintf("%dS", seconds))
		} else {
			frac := fmt.Sprintf("%09d", nanos)
			frac = strings.TrimRight(frac, "0")
			sb.WriteString(fmt.Sprintf("%d.%sS", seconds, frac))
		}
	}
	return sb.String(), nil
}
