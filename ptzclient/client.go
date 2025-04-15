// Package ptzclient implements a model to use the ONVIF PTZ service.
package ptzclient

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/hexbabe/sean-onvif"
	"github.com/hexbabe/sean-onvif/media"
	"github.com/hexbabe/sean-onvif/ptz"
	"github.com/hexbabe/sean-onvif/xsd"
	onvifxsd "github.com/hexbabe/sean-onvif/xsd/onvif"
	"github.com/viam-modules/viamrtsp"
	"go.viam.com/rdk/components/generic"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
)

const (
	defaultPanSpeed  = 0.5
	defaultTiltSpeed = 0.5
	defaultZoomSpeed = 0.5
)

// Model is the model for the ONVIF PTZ client.
var Model = viamrtsp.Family.WithModel("onvif-ptz-client")

func init() {
	resource.RegisterComponent(
		generic.API,
		Model,
		resource.Registration[resource.Resource, *Config]{
			Constructor: newOnvifPtzClientClient,
		},
	)
}

// Config represents the configuration for the ONVIF PTZ client.
type Config struct {
	Address      string `json:"address"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	ProfileToken string `json:"profile_token"`
}

// Validate validates the configuration for the ONVIF PTZ client.
func (cfg *Config) Validate(path string) ([]string, error) {
	if cfg.Address == "" {
		return nil, fmt.Errorf(`expected "address" attribute for %s %q`, Model.String(), path)
	}

	if cfg.Username == "" {
		return nil, fmt.Errorf(`expected "username" attribute for %s %q`, Model.String(), path)
	}
	if cfg.Password == "" {
		return nil, fmt.Errorf(`expected "password" attribute for %s %q`, Model.String(), path)
	}
	return nil, nil
}

type onvifPtzClient struct {
	resource.AlwaysRebuild

	name   resource.Name
	logger logging.Logger
	cfg    *Config
	dev    *onvif.Device

	cancelCtx  context.Context
	cancelFunc func()
}

func newOnvifPtzClientClient(
	ctx context.Context,
	deps resource.Dependencies,
	rawConf resource.Config,
	logger logging.Logger,
) (resource.Resource, error) {
	conf, err := resource.NativeConfig[*Config](rawConf)
	if err != nil {
		return nil, err
	}

	return NewClient(ctx, deps, rawConf.ResourceName(), conf, logger)
}

// NewClient creates a new ONVIF PTZ client.
func NewClient(
	_ context.Context,
	_ resource.Dependencies,
	name resource.Name,
	conf *Config,
	logger logging.Logger,
) (resource.Resource, error) {
	cancelCtx, cancelFunc := context.WithCancel(context.Background())

	logger.Debugf("Attempting to connect to ONVIF device at %s", conf.Address)
	dev, err := onvif.NewDevice(onvif.DeviceParams{
		Xaddr:    conf.Address,
		Username: conf.Username,
		Password: conf.Password,
	})
	if err != nil {
		cancelFunc()
		return nil, fmt.Errorf("failed to create ONVIF device for %s: %w", conf.Address, err)
	}
	logger.Info("Successfully connected to ONVIF device.")

	s := &onvifPtzClient{
		name:       name,
		logger:     logger,
		cfg:        conf,
		dev:        dev,
		cancelCtx:  cancelCtx,
		cancelFunc: cancelFunc,
	}
	if s.cfg.ProfileToken == "" {
		logger.Warn("No 'profile_token' configured. PTZ commands may fail. Run 'get-profiles' to discover available profiles.")
	}

	return s, nil
}

func (s *onvifPtzClient) Name() resource.Name {
	return s.name
}

// handleGetProfiles retrieves available media profiles from the camera and implements the get-profiles command logic.
func (s *onvifPtzClient) handleGetProfiles() (map[string]interface{}, error) {
	s.logger.Debug("Fetching media profiles...")
	req := media.GetProfiles{}
	res, err := s.dev.CallMethod(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call GetProfiles: %w", err)
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read GetProfiles response body: %w", err)
	}

	var envelope ProfilesEnvelope
	err = xml.Unmarshal(bodyBytes, &envelope)
	if err != nil {
		s.logger.Warnf("Failed to unmarshal GetProfiles response. Raw XML:\n%s", string(bodyBytes))
		return nil, fmt.Errorf("failed to unmarshal GetProfiles response: %w", err)
	}

	var tokens []string
	for _, p := range envelope.Body.GetProfilesResponse.Profiles {
		tokens = append(tokens, p.Token)
	}
	s.logger.Debugf("Found profiles: %v", tokens)
	return map[string]interface{}{"profiles": tokens}, nil
}

// handleGetStatus implements the get-status command logic.
func (s *onvifPtzClient) handleGetStatus() (map[string]interface{}, error) {
	if s.cfg.ProfileToken == "" {
		return nil, errors.New("profile_token is not configured for this component")
	}
	profileToken := onvifxsd.ReferenceToken(s.cfg.ProfileToken)

	req := ptz.GetStatus{ProfileToken: profileToken}
	s.logger.Debugf("Sending GetStatus request for profile: %s", profileToken)

	res, err := s.dev.CallMethod(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call GetStatus: %w", err)
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read GetStatus response body: %w", err)
	}

	var statusEnvelope CustomGetStatusEnvelope
	err = xml.Unmarshal(bodyBytes, &statusEnvelope)
	if err != nil {
		s.logger.Warnf("Failed to unmarshal GetStatus response using custom structs. Raw XML:\n%s", string(bodyBytes))
		return nil, fmt.Errorf("failed to unmarshal GetStatus response: %w", err)
	}

	ptzStatus := statusEnvelope.Body.GetResponse.PTZStatus

	return map[string]interface{}{
		"position": map[string]interface{}{
			"pan_tilt": map[string]interface{}{
				"x":     ptzStatus.Position.PanTilt.X,
				"y":     ptzStatus.Position.PanTilt.Y,
				"space": ptzStatus.Position.PanTilt.Space,
			},
			"zoom": map[string]interface{}{
				"x":     ptzStatus.Position.Zoom.X,
				"space": ptzStatus.Position.Zoom.Space,
			},
		},
		"move_status": map[string]interface{}{
			"pan_tilt": ptzStatus.MoveStatus.PanTilt,
			"zoom":     ptzStatus.MoveStatus.Zoom,
		},
		"utc_time": ptzStatus.UtcTime,
	}, nil
}

// handleStop implements the stop command logic.
func (s *onvifPtzClient) handleStop(cmd map[string]interface{}) (map[string]interface{}, error) {
	if s.cfg.ProfileToken == "" {
		return nil, errors.New("profile_token is not configured for this component")
	}
	profileToken := onvifxsd.ReferenceToken(s.cfg.ProfileToken)

	stopPanTilt := getOptionalBool(cmd, "pan_tilt", true)
	stopZoom := getOptionalBool(cmd, "zoom", true)

	req := ptz.Stop{
		ProfileToken: profileToken,
		PanTilt:      xsd.Boolean(stopPanTilt),
		Zoom:         xsd.Boolean(stopZoom),
	}

	s.logger.Debugf("Sending Stop command (PanTilt: %v, Zoom: %v) for profile %s...", stopPanTilt, stopZoom, profileToken)
	_, err := s.dev.CallMethod(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Stop: %w", err)
	}
	s.logger.Infof("Stop command sent successfully for profile %s.", profileToken)
	return map[string]interface{}{"success": true}, nil
}

// handleContinuousMove implements the continuous-move command logic.
func (s *onvifPtzClient) handleContinuousMove(cmd map[string]interface{}) (map[string]interface{}, error) {
	if s.cfg.ProfileToken == "" {
		return nil, errors.New("profile_token is not configured for this component")
	}
	profileToken := onvifxsd.ReferenceToken(s.cfg.ProfileToken)

	panSpeed := getOptionalFloat64(cmd, "pan_speed", 0.0)
	tiltSpeed := getOptionalFloat64(cmd, "tilt_speed", 0.0)
	zoomSpeed := getOptionalFloat64(cmd, "zoom_speed", 0.0)

	if panSpeed < -1.0 || panSpeed > 1.0 || tiltSpeed < -1.0 || tiltSpeed > 1.0 || zoomSpeed < -1.0 || zoomSpeed > 1.0 {
		return nil, errors.New("speed values (pan_speed, tilt_speed, zoom_speed) must be between -1.0 and 1.0")
	}

	req := ptz.ContinuousMove{
		ProfileToken: profileToken,
		Velocity: onvifxsd.PTZSpeed{
			PanTilt: onvifxsd.Vector2D{
				X:     panSpeed,
				Y:     tiltSpeed,
				Space: ContinuousPanTiltVelocityGenericSpace,
			},
			Zoom: onvifxsd.Vector1D{
				X:     zoomSpeed,
				Space: ContinuousZoomVelocityGenericSpace,
			},
		},
	}

	s.logger.Debugf(
		"Sending ContinuousMove (PanSpeed: %.2f, TiltSpeed: %.2f, ZoomSpeed: %.2f) for profile %s...",
		panSpeed, tiltSpeed, zoomSpeed, profileToken,
	)
	_, err := s.dev.CallMethod(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call ContinuousMove: %w", err)
	}

	s.logger.Infof("ContinuousMove command sent successfully for profile %s. Send 'stop' command to halt.", profileToken)
	return map[string]interface{}{"success": true}, nil
}

// handleRelativeMove implements the relative-move command logic.
func (s *onvifPtzClient) handleRelativeMove(cmd map[string]interface{}) (map[string]interface{}, error) {
	if s.cfg.ProfileToken == "" {
		return nil, errors.New("profile_token is not configured for this component")
	}
	profileToken := onvifxsd.ReferenceToken(s.cfg.ProfileToken)

	panRelative := getOptionalFloat64(cmd, "pan", 0.0)
	tiltRelative := getOptionalFloat64(cmd, "tilt", 0.0)
	zoomRelative := getOptionalFloat64(cmd, "zoom", 0.0)
	useDegrees := getOptionalBool(cmd, "degrees", false)

	speedX := getOptionalFloat64(cmd, "speed_pan", defaultPanSpeed)
	speedY := getOptionalFloat64(cmd, "speed_tilt", defaultTiltSpeed)
	speedZ := getOptionalFloat64(cmd, "speed_zoom", defaultZoomSpeed)
	useSpeed := getOptionalBool(cmd, "use_speed", false)

	// Input validation based on degrees input
	if useDegrees {
		if panRelative < -180.0 || panRelative > 180.0 {
			return nil, errors.New("relative pan must be between -180.0 and 180.0 when using degrees")
		}
		if tiltRelative < -90.0 || tiltRelative > 90.0 {
			return nil, errors.New("relative tilt must be between -90.0 and 90.0 when using degrees")
		}
	} else {
		if panRelative < -1.0 || panRelative > 1.0 {
			return nil, errors.New("relative pan must be between -1.0 and 1.0 (use degrees=true for degrees)")
		}
		if tiltRelative < -1.0 || tiltRelative > 1.0 {
			return nil, errors.New("relative tilt must be between -1.0 and 1.0 (use degrees=true for degrees)")
		}
	}
	if zoomRelative < -1.0 || zoomRelative > 1.0 {
		return nil, errors.New("relative zoom must be between -1.0 and 1.0")
	}

	panTiltVector := onvifxsd.Vector2D{
		X: panRelative,
		Y: tiltRelative,
	}
	if useDegrees {
		panTiltVector.Space = RelativePanTiltTranslationSphericalDegrees
		s.logger.Debug("Using Spherical Degrees space for relative Pan/Tilt.")
	} else {
		panTiltVector.Space = RelativePanTiltTranslationGenericSpace
		s.logger.Debug("Using Generic Normalized space for relative Pan/Tilt.")
	}

	req := ptz.RelativeMove{
		ProfileToken: profileToken,
		Translation: onvifxsd.PTZVector{
			PanTilt: panTiltVector,
			Zoom: onvifxsd.Vector1D{
				X:     zoomRelative,
				Space: RelativeZoomTranslationGenericSpace, // zoom always generic space for relative
			},
		},
	}

	if useSpeed {
		if speedX < -1.0 || speedX > 1.0 || speedY < -1.0 || speedY > 1.0 || speedZ < -1.0 || speedZ > 1.0 {
			return nil, errors.New("speed values must be between -1.0 and 1.0")
		}
		req.Speed = onvifxsd.PTZSpeed{
			PanTilt: onvifxsd.Vector2D{X: speedX, Y: speedY},
			Zoom:    onvifxsd.Vector1D{X: speedZ},
		}
		s.logger.Debugf("Sending RelativeMove (P: %.3f, T: %.3f, Z: %.3f) with Speed (X: %.2f, Y: %.2f, Z: %.2f) for profile %s...",
			panRelative, tiltRelative, zoomRelative, speedX, speedY, speedZ, profileToken)
	} else {
		s.logger.Debugf("Sending RelativeMove (P: %.3f, T: %.3f, Z: %.3f) with default speed for profile %s...",
			panRelative, tiltRelative, zoomRelative, profileToken)
	}

	_, err := s.dev.CallMethod(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call RelativeMove: %w", err)
	}
	s.logger.Infof("RelativeMove command sent successfully for profile %s.", profileToken)
	return map[string]interface{}{"success": true}, nil
}

// handleAbsoluteMove implements the absolute-move command logic.
func (s *onvifPtzClient) handleAbsoluteMove(cmd map[string]interface{}) (map[string]interface{}, error) {
	if s.cfg.ProfileToken == "" {
		return nil, errors.New("profile_token is not configured for this component")
	}
	profileToken := onvifxsd.ReferenceToken(s.cfg.ProfileToken)

	panPos, err := getFloat64(cmd, "pan_position")
	if err != nil {
		return nil, err
	}
	tiltPos, err := getFloat64(cmd, "tilt_position")
	if err != nil {
		return nil, err
	}
	zoomPos, err := getFloat64(cmd, "zoom_position")
	if err != nil {
		return nil, err
	}
	panSpeed := getOptionalFloat64(cmd, "pan_speed", 1.0)
	tiltSpeed := getOptionalFloat64(cmd, "tilt_speed", 1.0)
	zoomSpeed := getOptionalFloat64(cmd, "zoom_speed", 1.0)

	// Validate speed ranges
	if panSpeed < 0 || panSpeed > 1.0 || tiltSpeed < 0 || tiltSpeed > 1.0 || zoomSpeed < 0 || zoomSpeed > 1.0 {
		return nil, errors.New("speed values (pan_speed, tilt_speed, zoom_speed) must be between 0.0 and 1.0 for absolute move")
	}

	req := ptz.AbsoluteMove{
		ProfileToken: profileToken,
		Position: onvifxsd.PTZVector{
			PanTilt: onvifxsd.Vector2D{
				X:     panPos,
				Y:     tiltPos,
				Space: AbsolutePanTiltPositionGenericSpace,
			},
			Zoom: onvifxsd.Vector1D{
				X:     zoomPos,
				Space: AbsoluteZoomPositionGenericSpace,
			},
		},
		Speed: onvifxsd.PTZSpeed{
			PanTilt: onvifxsd.Vector2D{
				X:     panSpeed,
				Y:     tiltSpeed,
				Space: AbsolutePanTiltPositionGenericSpace,
			},
			Zoom: onvifxsd.Vector1D{
				X:     zoomSpeed,
				Space: AbsoluteZoomPositionGenericSpace,
			},
		},
	}

	s.logger.Debugf("Sending AbsoluteMove (Pan: %.3f, Tilt: %.3f, Zoom: %.3f) with Speed (X: %.2f, Y: %.2f, Z: %.2f) for profile %s...",
		panPos, tiltPos, zoomPos, panSpeed, tiltSpeed, zoomSpeed, profileToken)

	_, err = s.dev.CallMethod(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call AbsoluteMove: %w", err)
	}
	s.logger.Infof("AbsoluteMove command sent successfully for profile %s.", profileToken)
	return map[string]interface{}{"success": true}, nil
}

// DoCommand maps incoming commands to the appropriate ONVIF PTZ action.
func (s *onvifPtzClient) DoCommand(_ context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	command, err := getString(cmd, "command")
	if err != nil {
		return nil, errors.New("invalid command request: 'command' key missing or not a string")
	}

	s.logger.Debugf("Received command: %s with args: %v", command, cmd)

	switch strings.ToLower(command) {
	case "get-profiles":
		return s.handleGetProfiles()
	case "get-status":
		return s.handleGetStatus()
	case "stop":
		return s.handleStop(cmd)
	case "continuous-move":
		return s.handleContinuousMove(cmd)
	case "relative-move":
		return s.handleRelativeMove(cmd)
	case "absolute-move":
		return s.handleAbsoluteMove(cmd)
	default:
		return nil, fmt.Errorf("unrecognized DoCommand command: %s", command)
	}
}

func (s *onvifPtzClient) Close(context.Context) error {
	_, err := s.handleStop(map[string]interface{}{"pan_tilt": true, "zoom": true})
	if err != nil {
		s.logger.Errorf("Failed to stop PTZ: %v", err)
	}
	s.cancelFunc()
	return nil
}
