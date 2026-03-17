package ptzclient

import (
	"context"
	"errors"
	"fmt"
	"math"

	commonpb "go.viam.com/api/common/v1"
	"go.viam.com/rdk/components/arm"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/referenceframe"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/spatialmath"

	"github.com/viam-modules/viamrtsp"
	"github.com/viam-modules/viamrtsp/viamonvif/ptz"
	"github.com/viam-modules/viamrtsp/viamonvif/xsd"
	"github.com/viam-modules/viamrtsp/viamonvif/xsd/onvif"
)

// PTZArmModel is the model for the ONVIF PTZ camera exposed as an arm component.
var PTZArmModel = viamrtsp.Family.WithModel("onvif-ptz")

func init() {
	resource.RegisterComponent(
		arm.API,
		PTZArmModel,
		resource.Registration[resource.Resource, *Config]{
			Constructor: newOnvifPtzArm,
		},
	)
}

func newOnvifPtzArm(
	ctx context.Context,
	deps resource.Dependencies,
	rawConf resource.Config,
	logger logging.Logger,
) (resource.Resource, error) {
	conf, err := resource.NativeConfig[*Config](rawConf)
	if err != nil {
		return nil, err
	}
	if _, ok := conf.Movements["absolute"]; !ok {
		return nil, errors.New(`onvif-ptz requires "movements.absolute.pan_tilt" to be configured`)
	}
	return NewClient(ctx, deps, rawConf.ResourceName(), conf, logger)
}

// --- framesystem.InputEnabled ---

// Kinematics returns the SVA kinematics model for this pan-tilt gimbal.
func (s *onvifPtzClient) Kinematics(_ context.Context) (referenceframe.Model, error) {
	b, err := buildSVA(s.cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build kinematics model: %w", err)
	}
	return referenceframe.UnmarshalModelJSON(b, s.name.ShortName())
}

// CurrentInputs returns the current pan/tilt joint positions in radians.
func (s *onvifPtzClient) CurrentInputs(ctx context.Context) ([]referenceframe.Input, error) {
	return s.JointPositions(ctx, nil)
}

// GoToInputs moves to the given joint positions in radians.
func (s *onvifPtzClient) GoToInputs(ctx context.Context, inputSteps ...[]referenceframe.Input) error {
	for _, inputs := range inputSteps {
		if err := s.MoveToJointPositions(ctx, inputs, nil); err != nil {
			return err
		}
	}
	return nil
}

// --- arm.Arm interface ---

// JointPositions returns the current pan/tilt as radians.
func (s *onvifPtzClient) JointPositions(_ context.Context, _ map[string]interface{}) ([]referenceframe.Input, error) {
	status, err := s.getRawStatus()
	if err != nil {
		return nil, err
	}
	abs := s.cfg.Movements["absolute"]
	pt := abs.PanTilt

	var panDeg, tiltDeg float64
	if pt.Space == AbsolutePanTiltPositionSphericalDegrees {
		panDeg = status.Position.PanTilt.X
		tiltDeg = status.Position.PanTilt.Y
	} else {
		panDeg = onvifToDegrees(status.Position.PanTilt.X, pt.XMin, pt.XMax, defaultPanMinDeg, defaultPanMaxDeg)
		tiltDeg = onvifToDegrees(status.Position.PanTilt.Y, pt.YMin, pt.YMax, defaultTiltMinDeg, defaultTiltMaxDeg)
	}

	return []referenceframe.Input{
		degreesToRadians(panDeg),
		degreesToRadians(tiltDeg),
	}, nil
}

// MoveToJointPositions commands the camera to a pan/tilt position given in radians.
func (s *onvifPtzClient) MoveToJointPositions(_ context.Context, positions []referenceframe.Input, _ map[string]interface{}) error {
	if len(positions) != 2 {
		return fmt.Errorf("expected 2 joint positions (pan, tilt), got %d", len(positions))
	}
	profileToken, err := s.profileToken()
	if err != nil {
		return err
	}

	abs := s.cfg.Movements["absolute"]
	pt := abs.PanTilt

	panDeg := radiansToDegrees(positions[0])
	tiltDeg := radiansToDegrees(positions[1])

	var panOnvif, tiltOnvif float64
	if pt.Space == AbsolutePanTiltPositionSphericalDegrees {
		panOnvif = panDeg
		tiltOnvif = tiltDeg
	} else {
		panOnvif = degreesToOnvif(panDeg, defaultPanMinDeg, defaultPanMaxDeg, pt.XMin, pt.XMax)
		tiltOnvif = degreesToOnvif(tiltDeg, defaultTiltMinDeg, defaultTiltMaxDeg, pt.YMin, pt.YMax)
	}

	_, err = s.callPTZMethod(ptz.AbsoluteMove{
		ProfileToken: profileToken,
		Position: onvif.PTZVector{
			PanTilt: onvif.Vector2D{X: panOnvif, Y: tiltOnvif, Space: xsd.AnyURI(AbsolutePanTiltPositionGenericSpace)},
		},
	}, nil)
	return err
}

// MoveThroughJointPositions executes a sequence of MoveToJointPositions calls.
func (s *onvifPtzClient) MoveThroughJointPositions(
	ctx context.Context,
	positions [][]referenceframe.Input,
	_ *arm.MoveOptions,
	extra map[string]interface{},
) error {
	for _, pos := range positions {
		if err := s.MoveToJointPositions(ctx, pos, extra); err != nil {
			return err
		}
	}
	return nil
}

// EndPosition returns the current gaze pose via forward kinematics.
func (s *onvifPtzClient) EndPosition(ctx context.Context, extra map[string]interface{}) (spatialmath.Pose, error) {
	inputs, err := s.JointPositions(ctx, extra)
	if err != nil {
		return nil, err
	}
	model, err := s.Kinematics(ctx)
	if err != nil {
		return nil, err
	}
	return model.Transform(inputs)
}

// MoveToPosition points the camera at the given pose using analytical IK.
// Only the position (X, Y, Z) of the pose is used; orientation is ignored.
func (s *onvifPtzClient) MoveToPosition(ctx context.Context, pose spatialmath.Pose, extra map[string]interface{}) error {
	pt := pose.Point()
	pan := math.Atan2(pt.Y, pt.X)
	tilt := math.Atan2(-pt.Z, math.Sqrt(pt.X*pt.X+pt.Y*pt.Y))
	return s.MoveToJointPositions(ctx, []referenceframe.Input{pan, tilt}, extra)
}

// Stop halts all PTZ movement.
func (s *onvifPtzClient) Stop(_ context.Context, _ map[string]interface{}) error {
	_, err := s.handleStop(map[string]interface{}{})
	return err
}

// IsMoving returns true if the camera is currently panning or tilting.
func (s *onvifPtzClient) IsMoving(_ context.Context) (bool, error) {
	status, err := s.getRawStatus()
	if err != nil {
		return false, err
	}
	return status.MoveStatus.PanTilt == "MOVING", nil
}

// Geometries is not implemented for PTZ cameras.
func (s *onvifPtzClient) Geometries(_ context.Context, _ map[string]interface{}) ([]spatialmath.Geometry, error) {
	return nil, nil
}

// Get3DModels is not implemented for PTZ cameras.
func (s *onvifPtzClient) Get3DModels(_ context.Context, _ map[string]interface{}) (map[string]*commonpb.Mesh, error) {
	return nil, nil
}

// getRawStatus calls ONVIF GetStatus and returns the parsed PTZ status.
func (s *onvifPtzClient) getRawStatus() (CustomPTZStatus, error) {
	profileToken, err := s.profileToken()
	if err != nil {
		return CustomPTZStatus{}, err
	}
	var envelope CustomGetStatusEnvelope
	if _, err := s.callPTZMethod(ptz.GetStatus{ProfileToken: profileToken}, &envelope); err != nil {
		return CustomPTZStatus{}, fmt.Errorf("GetStatus failed: %w", err)
	}
	return envelope.Body.GetResponse.PTZStatus, nil
}
