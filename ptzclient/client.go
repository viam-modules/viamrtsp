// Package ptzclient implements a model to use the ONVIF PTZ service.
package ptzclient

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"github.com/viam-modules/viamrtsp"
	"github.com/viam-modules/viamrtsp/viamonvif/device"
	"github.com/viam-modules/viamrtsp/viamonvif/ptz"
	"github.com/viam-modules/viamrtsp/viamonvif/xsd"
	"github.com/viam-modules/viamrtsp/viamonvif/xsd/onvif"
	"go.viam.com/rdk/components/generic"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
)

const (
	defaultPanSpeed              = 0.5
	defaultTiltSpeed             = 0.5
	defaultZoomSpeed             = 0.5
	maxContinuousMovementTimeout = "PT10S"
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

// PanTiltSpace defines the pan and tilt space for the PTZ movement.
type PanTiltSpace struct {
	XMin  float64 `json:"x_min"`
	XMax  float64 `json:"x_max"`
	YMin  float64 `json:"y_min"`
	YMax  float64 `json:"y_max"`
	Space string  `json:"space"`
}

// ZoomSpace defines the zoom space for the PTZ movement.
type ZoomSpace struct {
	XMin  float64 `json:"x_min"`
	XMax  float64 `json:"x_max"`
	Space string  `json:"space"`
}

// PTZMovement defines the movement parameters for pan, tilt, and zoom.
type PTZMovement struct {
	PanTilt PanTiltSpace `json:"pan_tilt,omitempty"`
	Zoom    ZoomSpace    `json:"zoom,omitempty"`
}

// Config represents the configuration for the ONVIF PTZ client.
type Config struct {
	Address      string                 `json:"address"`
	Username     string                 `json:"username,omitempty"`
	Password     string                 `json:"password,omitempty"`
	ProfileToken string                 `json:"profile_token"`
	NodeToken    string                 `json:"ptz_node_token,omitempty"`
	Movements    map[string]PTZMovement `json:"movements,omitempty"`
	DiscoveryDep string                 `json:"discovery_dep,omitempty"`
	RTSPAddress  string                 `json:"rtsp_address,omitempty"`
}

// Validate validates the configuration for the ONVIF PTZ client.
func (cfg *Config) Validate(path string) ([]string, []string, error) {
	if cfg.Address == "" {
		return nil, nil, fmt.Errorf(`expected "address" attribute for %s %q`, Model.String(), path)
	}

	return nil, nil, nil
}

type onvifPtzClient struct {
	resource.AlwaysRebuild

	name   resource.Name
	logger logging.Logger
	cfg    *Config
	dev    *device.Device

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
	ctx context.Context,
	_ resource.Dependencies,
	name resource.Name,
	conf *Config,
	logger logging.Logger,
) (resource.Resource, error) {
	cancelCtx, cancelFunc := context.WithCancel(context.Background())

	logger.Debugf("Attempting to connect to ONVIF device at %s", conf.Address)

	xaddr, err := url.Parse(conf.Address)
	if err != nil {
		cancelFunc()
		return nil, fmt.Errorf("failed to parse ONVIF address %s: %w", conf.Address, err)
	}

	params := device.Params{
		Xaddr:                    xaddr,
		SkipLocalTLSVerification: true,
	}
	// Credentials are optional for unauthenticated cameras
	if conf.Username != "" {
		params.Username = conf.Username
	}
	if conf.Password != "" {
		params.Password = conf.Password
	}
	dev, err := device.NewDevice(ctx, params, logger)
	if err != nil {
		cancelFunc()
		return nil, fmt.Errorf("failed to create ONVIF device for %s: %w", conf.Address, err)
	}
	logger.Info("successfully connected to ONVIF device.")

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

// callPTZMethod calls an ONVIF PTZ method and unmarshals the response into result.
// If result is nil, unmarshaling is skipped and raw bytes are returned.
func (s *onvifPtzClient) callPTZMethod(req interface{}, result interface{}) ([]byte, error) {
	bodyBytes, err := s.dev.CallPTZMethod(s.cancelCtx, req)
	if err != nil {
		return nil, err
	}
	s.logger.Debugf("%s raw response: %s", reflect.TypeOf(req).Name(), string(bodyBytes))

	if result != nil {
		if err := xml.Unmarshal(bodyBytes, result); err != nil {
			s.logger.Warnf("Unmarshal failed. Raw XML:\n%s", string(bodyBytes))
			return nil, fmt.Errorf("failed to unmarshal: %w", err)
		}
	}
	return bodyBytes, nil
}

// profileToken returns the configured profile token or an error if not set.
func (s *onvifPtzClient) profileToken() (onvif.ReferenceToken, error) {
	if s.cfg.ProfileToken == "" {
		return "", errors.New("profile_token is not configured for this component")
	}
	return onvif.ReferenceToken(s.cfg.ProfileToken), nil
}

// handleGetProfiles retrieves available media profiles from the camera and implements the get-profiles command logic.
func (s *onvifPtzClient) handleGetProfiles() (map[string]interface{}, error) {
	s.logger.Debug("Fetching media profiles...")

	resp, err := s.dev.GetProfiles(s.cancelCtx)
	if err != nil {
		return nil, fmt.Errorf("get profiles failed: %w", err)
	}

	var tokens []string
	for _, p := range resp.Profiles {
		tokens = append(tokens, string(p.Token))
	}
	s.logger.Debugf("Found profiles: %v", tokens)
	return map[string]interface{}{"profiles": tokens}, nil
}

func (s *onvifPtzClient) handleGetServiceCapabilities() (map[string]interface{}, error) {
	s.logger.Debug("Sending GetServiceCapabilities request")

	var envelope GetServiceCapabilitiesEnvelope
	if _, err := s.callPTZMethod(ptz.GetServiceCapabilities{}, &envelope); err != nil {
		return nil, fmt.Errorf("get service capabilities failed: %w", err)
	}

	caps := envelope.Body.GetServiceCapabilitiesResponse.Capabilities
	return map[string]interface{}{
		"service_capabilities": map[string]interface{}{
			"EFlip":                       bool(caps.EFlip),
			"Reverse":                     bool(caps.Reverse),
			"GetCompatibleConfigurations": bool(caps.GetCompatibleConfigurations),
			"MoveStatus":                  bool(caps.MoveStatus),
			"StatusPosition":              bool(caps.StatusPosition),
		},
	}, nil
}

// handleGetConfiguration implements the get-configuration command logic.
func (s *onvifPtzClient) handleGetConfiguration() (map[string]interface{}, error) {
	profileToken, err := s.profileToken()
	if err != nil {
		return nil, err
	}

	var envelope GetConfigurationEnvelope
	if _, err := s.callPTZMethod(ptz.GetConfiguration{ProfileToken: profileToken}, &envelope); err != nil {
		return nil, fmt.Errorf("get configuration failed: %w", err)
	}

	return map[string]interface{}{"configuration": envelope.Body.GetConfigurationResponse.PTZConfiguration}, nil
}

// handleGetConfigurations implements the get-configurations command logic.
func (s *onvifPtzClient) handleGetConfigurations() (map[string]interface{}, error) {
	if _, err := s.profileToken(); err != nil {
		return nil, err
	}

	s.logger.Debug("Sending GetConfigurations request")

	var envelope GetConfigurationsEnvelope
	if _, err := s.callPTZMethod(ptz.GetConfigurations{}, &envelope); err != nil {
		return nil, fmt.Errorf("get configurations failed: %w", err)
	}

	return map[string]interface{}{"configurations": envelope.Body.GetConfigurationsResponse.PTZConfiguration}, nil
}

// handleGetStatus implements the get-status command logic.
func (s *onvifPtzClient) handleGetStatus() (map[string]interface{}, error) {
	profileToken, err := s.profileToken()
	if err != nil {
		return nil, err
	}

	s.logger.Debugf("Sending GetStatus request for profile: %s", profileToken)

	var envelope CustomGetStatusEnvelope
	if _, err := s.callPTZMethod(ptz.GetStatus{ProfileToken: profileToken}, &envelope); err != nil {
		return nil, fmt.Errorf("get status failed: %w", err)
	}

	status := envelope.Body.GetResponse.PTZStatus
	return map[string]interface{}{
		"position": map[string]interface{}{
			"pan_tilt": map[string]interface{}{
				"x":     status.Position.PanTilt.X,
				"y":     status.Position.PanTilt.Y,
				"space": status.Position.PanTilt.Space,
			},
			"zoom": map[string]interface{}{
				"x":     status.Position.Zoom.X,
				"space": status.Position.Zoom.Space,
			},
		},
		"move_status": map[string]interface{}{
			"pan_tilt": status.MoveStatus.PanTilt,
			"zoom":     status.MoveStatus.Zoom,
		},
		"utc_time": status.UtcTime,
	}, nil
}

// handleStop implements the stop command logic.
func (s *onvifPtzClient) handleStop(cmd map[string]interface{}) (map[string]interface{}, error) {
	profileToken, err := s.profileToken()
	if err != nil {
		return nil, err
	}

	stopPanTilt := getOptionalBool(cmd, "pan_tilt", true)
	stopZoom := getOptionalBool(cmd, "zoom", true)

	s.logger.Debugf("Sending Stop command (PanTilt: %v, Zoom: %v) for profile %s...", stopPanTilt, stopZoom, profileToken)

	bodyBytes, err := s.callPTZMethod(ptz.Stop{
		ProfileToken: profileToken,
		PanTilt:      xsd.Boolean(stopPanTilt),
		Zoom:         xsd.Boolean(stopZoom),
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("stop failed: %w", err)
	}
	return map[string]interface{}{"response": string(bodyBytes)}, nil
}

// handleContinuousMove implements the continuous-move command logic.
func (s *onvifPtzClient) handleContinuousMove(cmd map[string]interface{}) (map[string]interface{}, error) {
	profileToken, err := s.profileToken()
	if err != nil {
		return nil, err
	}

	panSpeed := getOptionalFloat64(cmd, "pan_speed", 0.0)
	tiltSpeed := getOptionalFloat64(cmd, "tilt_speed", 0.0)
	zoomSpeed := getOptionalFloat64(cmd, "zoom_speed", 0.0)
	timeout := getOptionalDuration(cmd, "timeout", maxContinuousMovementTimeout)

	if err := validateSpeeds(panSpeed, tiltSpeed, zoomSpeed, true); err != nil {
		return nil, err
	}

	s.logger.Debugf("Sending ContinuousMove (Pan: %.2f, Tilt: %.2f, Zoom: %.2f, Timeout: %s) for profile %s...",
		panSpeed, tiltSpeed, zoomSpeed, timeout, profileToken)

	bodyBytes, err := s.callPTZMethod(ptz.ContinuousMove{
		ProfileToken: profileToken,
		Velocity: onvif.PTZSpeed{
			PanTilt: onvif.Vector2D{X: panSpeed, Y: tiltSpeed, Space: xsd.AnyURI(ContinuousPanTiltVelocityGenericSpace)},
			Zoom:    onvif.Vector1D{X: zoomSpeed, Space: xsd.AnyURI(ContinuousZoomVelocityGenericSpace)},
		},
		Timeout: xsd.Duration(timeout),
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("continuous move failed: %w", err)
	}
	return map[string]interface{}{"response": string(bodyBytes)}, nil
}

// handleRelativeMove implements the relative-move command logic.
func (s *onvifPtzClient) handleRelativeMove(cmd map[string]interface{}) (map[string]interface{}, error) {
	profileToken, err := s.profileToken()
	if err != nil {
		return nil, err
	}

	panRelative := getOptionalFloat64(cmd, "pan", 0.0)
	tiltRelative := getOptionalFloat64(cmd, "tilt", 0.0)
	zoomTranslation := getOptionalFloat64(cmd, "zoom", 0.0)
	if _, ok := cmd["zoom"]; !ok {
		zoomTranslation = getOptionalFloat64(cmd, "zoom_translation", 0.0)
	}

	panTiltSpace := RelativePanTiltTranslationGenericSpace
	if space, ok := getOptionalString(cmd, "pan_tilt_space"); ok && space != "" {
		panTiltSpace = space
		s.logger.Debugf("Using custom space for relative Pan/Tilt: %s", panTiltSpace)
	} else if useDegrees := getOptionalBool(cmd, "degrees", false); useDegrees {
		panTiltSpace = RelativePanTiltTranslationSphericalDegrees
		s.logger.Debug("Using Spherical Degrees space for relative Pan/Tilt.")
	} else {
		s.logger.Debug("Using Generic Normalized space for relative Pan/Tilt.")
	}

	req := ptz.RelativeMove{
		ProfileToken: profileToken,
		Translation: onvif.PTZVector{
			PanTilt: onvif.Vector2D{X: panRelative, Y: tiltRelative, Space: xsd.AnyURI(panTiltSpace)},
			Zoom:    onvif.Vector1D{X: zoomTranslation, Space: xsd.AnyURI(RelativeZoomTranslationGenericSpace)},
		},
	}

	spd := extractSpeeds(cmd)
	if spd.provided {
		if err := validateSpeeds(spd.pan, spd.tilt, spd.zoom, false); err != nil {
			return nil, err
		}
		req.Speed = onvif.PTZSpeed{
			PanTilt: onvif.Vector2D{X: spd.pan, Y: spd.tilt, Space: xsd.AnyURI(panTiltSpace)},
			Zoom:    onvif.Vector1D{X: spd.zoom, Space: xsd.AnyURI(RelativeZoomTranslationGenericSpace)},
		}
		s.logger.Debugf("Sending RelativeMove (P: %.3f, T: %.3f, Z: %.3f) with Speed (%.2f, %.2f, %.2f) for profile %s...",
			panRelative, tiltRelative, zoomTranslation, spd.pan, spd.tilt, spd.zoom, profileToken)
	} else {
		s.logger.Debugf("Sending RelativeMove (P: %.3f, T: %.3f, Z: %.3f) using camera default speed for profile %s...",
			panRelative, tiltRelative, zoomTranslation, profileToken)
	}

	bodyBytes, err := s.callPTZMethod(req, nil)
	if err != nil {
		return nil, fmt.Errorf("relative move failed: %w", err)
	}
	return map[string]interface{}{"response": string(bodyBytes)}, nil
}

// handleAbsoluteMove implements the absolute-move command logic.
func (s *onvifPtzClient) handleAbsoluteMove(cmd map[string]interface{}) (map[string]interface{}, error) {
	profileToken, err := s.profileToken()
	if err != nil {
		return nil, err
	}

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

	// Validate position ranges (-1.0 to 1.0 for pan/tilt, 0.0 to 1.0 for zoom)
	if panPos < -1.0 || panPos > 1.0 || tiltPos < -1.0 || tiltPos > 1.0 || zoomPos < 0.0 || zoomPos > 1.0 {
		return nil, errors.New("position values must be within normalized range (-1 to 1 for pan/tilt, 0 to 1 for zoom)")
	}

	req := ptz.AbsoluteMove{
		ProfileToken: profileToken,
		Position: onvif.PTZVector{
			PanTilt: onvif.Vector2D{X: panPos, Y: tiltPos, Space: xsd.AnyURI(AbsolutePanTiltPositionGenericSpace)},
			Zoom:    onvif.Vector1D{X: zoomPos, Space: xsd.AnyURI(AbsoluteZoomPositionGenericSpace)},
		},
	}

	spd := extractSpeeds(cmd)
	if spd.provided {
		if err := validateSpeeds(spd.pan, spd.tilt, spd.zoom, false); err != nil {
			return nil, err
		}
		req.Speed = onvif.PTZSpeed{
			PanTilt: onvif.Vector2D{X: spd.pan, Y: spd.tilt, Space: xsd.AnyURI(AbsolutePanTiltPositionGenericSpace)},
			Zoom:    onvif.Vector1D{X: spd.zoom, Space: xsd.AnyURI(AbsoluteZoomPositionGenericSpace)},
		}
		s.logger.Debugf("Sending AbsoluteMove (P: %.3f, T: %.3f, Z: %.3f) with Speed (%.2f, %.2f, %.2f) for profile %s...",
			panPos, tiltPos, zoomPos, spd.pan, spd.tilt, spd.zoom, profileToken)
	} else {
		s.logger.Debugf("Sending AbsoluteMove (P: %.3f, T: %.3f, Z: %.3f) using camera default speed for profile %s...",
			panPos, tiltPos, zoomPos, profileToken)
	}

	bodyBytes, err := s.callPTZMethod(req, nil)
	if err != nil {
		return nil, fmt.Errorf("absolute move failed: %w", err)
	}
	return map[string]interface{}{"response": string(bodyBytes)}, nil
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
	case "get-configurations":
		return s.handleGetConfigurations()
	case "get-configuration":
		return s.handleGetConfiguration()
	case "get-service-capabilities":
		return s.handleGetServiceCapabilities()
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
