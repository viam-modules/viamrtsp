// Package ptzclient implements a model to use the ONVIF PTZ service.
package ptzclient

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	onvif "github.com/hexbabe/sean-onvif"
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
	params := onvif.DeviceParams{Xaddr: conf.Address}
	// Credentials are optional for unauthenticated cameras
	if conf.Username != "" {
		params.Username = conf.Username
	}
	if conf.Password != "" {
		params.Password = conf.Password
	}
	dev, err := onvif.NewDevice(params)
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

// handleGetProfiles retrieves available media profiles from the camera and implements the get-profiles command logic.
func (s *onvifPtzClient) handleGetProfiles() (map[string]interface{}, error) {
	s.logger.Debug("Fetching media profiles...")
	req := media.GetProfiles{}
	res, err := s.dev.CallMethod(req, s.logger)
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

func (s *onvifPtzClient) handleGetServiceCapabilities() (map[string]interface{}, error) {
	req := ptz.GetServiceCapabilities{}
	s.logger.Debugf("Sending GetServiceCapabilities request")

	res, err := s.dev.CallMethod(req, s.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to call GetServiceCapabilities: %w", err)
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read GetServiceCapabilities response body: %w", err)
	}
	s.logger.Debugf("GetServiceCapabilities raw response body: %s", string(bodyBytes))

	var serviceCapabilitiesEnvelope ptz.GetServiceCapabilitiesResponse
	err = xml.Unmarshal(bodyBytes, &serviceCapabilitiesEnvelope)
	if err != nil {
		s.logger.Warnf("Failed to unmarshal GetServiceCapabilities response using custom structs. Raw XML:\n%s", string(bodyBytes))
		return nil, fmt.Errorf("failed to unmarshal GetServiceCapabilities response: %w", err)
	}

	serviceCapabilities := serviceCapabilitiesEnvelope.Capabilities
	// Convert xsd.Boolean values to Go bool for protobuf compatibility
	convertedCapabilities := map[string]interface{}{
		"EFlip":                       bool(serviceCapabilities.EFlip),
		"Reverse":                     bool(serviceCapabilities.Reverse),
		"GetCompatibleConfigurations": bool(serviceCapabilities.GetCompatibleConfigurations),
		"MoveStatus":                  bool(serviceCapabilities.MoveStatus),
		"StatusPosition":              bool(serviceCapabilities.StatusPosition),
	}
	return map[string]interface{}{"service_capabilities": convertedCapabilities}, nil
}

// handleGetConfiguration implements the get-configuration command logic.
func (s *onvifPtzClient) handleGetConfiguration(cmd map[string]interface{}) (map[string]interface{}, error) {
	if s.cfg.ProfileToken == "" {
		return nil, errors.New("profile_token is not configured for this component")
	}
	profileToken := onvifxsd.ReferenceToken(s.cfg.ProfileToken)
	req := ptz.GetConfiguration{ProfileToken: profileToken}
	res, err := s.dev.CallMethod(req, s.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to call GetConfiguration: %w", err)
	}
	defer res.Body.Close()
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read GetConfiguration response body: %w", err)
	}
	s.logger.Debugf("GetConfiguration raw response body: %s", string(bodyBytes))
	var configurationEnvelope ptz.GetConfigurationResponse
	err = xml.Unmarshal(bodyBytes, &configurationEnvelope)
	if err != nil {
		s.logger.Warnf("Failed to unmarshal GetConfiguration response using custom structs. Raw XML:\n%s", string(bodyBytes))
		return nil, fmt.Errorf("failed to unmarshal GetConfiguration response: %w", err)
	}
	configuration := configurationEnvelope.PTZConfiguration
	return map[string]interface{}{
		"configuration": configuration,
	}, nil
}

// handleGetConfigurations implements the get-configurations command logic.
func (s *onvifPtzClient) handleGetConfigurations() (map[string]interface{}, error) {
	if s.cfg.ProfileToken == "" {
		return nil, errors.New("profile_token is not configured for this component")
	}

	req := ptz.GetConfigurations{}
	s.logger.Debugf("Sending GetConfigurations request")

	res, err := s.dev.CallMethod(req, s.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to call GetConfigurations: %w", err)
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read GetConfigurations response body: %w", err)
	}
	s.logger.Debugf("GetConfigurations raw response body: %s", string(bodyBytes))

	var configurationsEnvelope ptz.GetConfigurationsResponse
	err = xml.Unmarshal(bodyBytes, &configurationsEnvelope)
	if err != nil {
		s.logger.Warnf("Failed to unmarshal GetConfigurations response using custom structs. Raw XML:\n%s", string(bodyBytes))
		return nil, fmt.Errorf("failed to unmarshal GetConfigurations response: %w", err)
	}

	configurations := configurationsEnvelope.PTZConfiguration

	return map[string]interface{}{
		"configurations": configurations,
	}, nil
}

// handleGetStatus implements the get-status command logic.
func (s *onvifPtzClient) handleGetStatus() (map[string]interface{}, error) {
	if s.cfg.ProfileToken == "" {
		return nil, errors.New("profile_token is not configured for this component")
	}
	profileToken := onvifxsd.ReferenceToken(s.cfg.ProfileToken)

	req := ptz.GetStatus{ProfileToken: profileToken}
	s.logger.Debugf("Sending GetStatus request for profile: %s", profileToken)

	res, err := s.dev.CallMethod(req, s.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to call GetStatus: %w", err)
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read GetStatus response body: %w", err)
	}
	s.logger.Debugf("GetStatus raw response body: %s", string(bodyBytes))

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
	res, err := s.dev.CallMethod(req, s.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to call Stop: %w", err)
	}
	defer res.Body.Close()

	bodyBytes, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read Stop response body: %w", readErr)
	}
	response := string(bodyBytes)
	s.logger.Debugf("Stop raw response body: %s", response)

	return map[string]interface{}{"response": response}, nil
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
	timeout := getOptionalDuration(cmd, "timeout", maxContinuousMovementTimeout)

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
		Timeout: xsd.Duration(timeout),
	}

	s.logger.Debugf(
		"Sending ContinuousMove (PanSpeed: %.2f, TiltSpeed: %.2f, ZoomSpeed: %.2f, Timeout: %s) for profile %s...",
		panSpeed, tiltSpeed, zoomSpeed, timeout, profileToken,
	)
	res, err := s.dev.CallMethod(req, s.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to call ContinuousMove: %w", err)
	}
	defer res.Body.Close()

	bodyBytes, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read ContinuousMove response body: %w", readErr)
	}
	response := string(bodyBytes)
	s.logger.Debugf("ContinuousMove raw response body: %s", response)

	return map[string]interface{}{"response": response}, nil
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

	zoomTranslation, err := getFloat64(cmd, "zoom_translation")
	if err != nil {
		return nil, err
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

	// Check if any speed parameters were provided by the user
	_, panSpeedProvided := cmd["pan_speed"]
	_, tiltSpeedProvided := cmd["tilt_speed"]
	_, zoomSpeedProvided := cmd["zoom_speed"]
	sendSpeed := panSpeedProvided || tiltSpeedProvided || zoomSpeedProvided

	// Get speed values only if we intend to send them, using defaults if necessary
	var panSpeed, tiltSpeed, zoomSpeed float64
	if sendSpeed {
		panSpeed = getOptionalFloat64(cmd, "pan_speed", defaultPanSpeed)
		tiltSpeed = getOptionalFloat64(cmd, "tilt_speed", defaultTiltSpeed)
		zoomSpeed = getOptionalFloat64(cmd, "zoom_speed", defaultZoomSpeed)
	}

	req := ptz.RelativeMove{
		ProfileToken: profileToken,
		Translation: onvifxsd.PTZVector{
			PanTilt: panTiltVector,
			Zoom: onvifxsd.Vector1D{
				X:     zoomTranslation,
				Space: RelativeZoomTranslationGenericSpace,
			},
		},
	}

	if sendSpeed {
		// Validate speeds if they are being sent
		if panSpeed < 0.0 || panSpeed > 1.0 || tiltSpeed < 0.0 || tiltSpeed > 1.0 || zoomSpeed < 0.0 || zoomSpeed > 1.0 {
			return nil, errors.New("speed values (pan_speed, tilt_speed, zoom_speed) must be between 0.0 and 1.0 for relative move")
		}
		req.Speed = onvifxsd.PTZSpeed{
			PanTilt: onvifxsd.Vector2D{
				X:     panSpeed,
				Y:     tiltSpeed,
				Space: panTiltVector.Space,
			},
			Zoom: onvifxsd.Vector1D{
				X:     zoomSpeed,
				Space: RelativeZoomTranslationGenericSpace,
			},
		}
		s.logger.Debugf("Sending RelativeMove (P: %.3f, T: %.3f, Z: %.3f) with Speed (X: %.2f, Y: %.2f, Z: %.2f) for profile %s...",
			panRelative, tiltRelative, zoomRelative, panSpeed, tiltSpeed, zoomSpeed, profileToken)
	} else {
		s.logger.Debugf("Sending RelativeMove (P: %.3f, T: %.3f, Z: %.3f) using camera default speed for profile %s...",
			panRelative, tiltRelative, zoomRelative, profileToken)
	}

	res, err := s.dev.CallMethod(req, s.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to call RelativeMove: %w", err)
	}
	defer res.Body.Close()

	bodyBytes, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read RelativeMove response body: %w", readErr)
	}
	response := string(bodyBytes)
	s.logger.Debugf("RelativeMove raw response body: %s", response)

	return map[string]interface{}{"response": response}, nil
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

	// Check if any speed parameters were provided
	_, panSpeedProvided := cmd["pan_speed"]
	_, tiltSpeedProvided := cmd["tilt_speed"]
	_, zoomSpeedProvided := cmd["zoom_speed"]
	sendSpeed := panSpeedProvided || tiltSpeedProvided || zoomSpeedProvided

	// Get speed values only if sending them (default 1.0)
	var panSpeed, tiltSpeed, zoomSpeed float64
	if sendSpeed {
		panSpeed = getOptionalFloat64(cmd, "pan_speed", defaultPanSpeed)
		tiltSpeed = getOptionalFloat64(cmd, "tilt_speed", defaultTiltSpeed)
		zoomSpeed = getOptionalFloat64(cmd, "zoom_speed", defaultZoomSpeed)
	}

	// Validate position ranges (-1.0 to 1.0 for pan/tilt, 0.0 to 1.0 for zoom)
	if panPos < -1.0 || panPos > 1.0 || tiltPos < -1.0 || tiltPos > 1.0 || zoomPos < 0.0 || zoomPos > 1.0 {
		return nil, errors.New("position values must be within normalized range (-1 to 1 for pan/tilt, 0 to 1 for zoom)")
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
	}

	if sendSpeed {
		// Validate speeds if they are being sent
		if panSpeed < 0.0 || panSpeed > 1.0 || tiltSpeed < 0.0 || tiltSpeed > 1.0 || zoomSpeed < 0.0 || zoomSpeed > 1.0 {
			return nil, errors.New("speed values (pan_speed, tilt_speed, zoom_speed) must be between 0.0 and 1.0 for absolute move")
		}
		req.Speed = onvifxsd.PTZSpeed{
			PanTilt: onvifxsd.Vector2D{
				X:     panSpeed,
				Y:     tiltSpeed,
				Space: AbsolutePanTiltPositionGenericSpace,
			},
			Zoom: onvifxsd.Vector1D{
				X:     zoomSpeed,
				Space: AbsoluteZoomPositionGenericSpace,
			},
		}
		s.logger.Debugf("Sending AbsoluteMove (P: %.3f, T: %.3f, Z: %.3f) with Speed (X: %.2f, Y: %.2f, Z: %.2f) for profile %s...",
			panPos, tiltPos, zoomPos, panSpeed, tiltSpeed, zoomSpeed, profileToken)
	} else {
		s.logger.Debugf("Sending AbsoluteMove (P: %.3f, T: %.3f, Z: %.3f) using camera default speed for profile %s...",
			panPos, tiltPos, zoomPos, profileToken)
	}

	res, err := s.dev.CallMethod(req, s.logger)
	if err != nil {
		// This is an HTTP or connection level error
		return nil, fmt.Errorf("failed to call AbsoluteMove: %w", err)
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read AbsoluteMove response body: %w", err)
	}

	// this could contain a response body error from the camera
	response := string(bodyBytes)
	s.logger.Debugf("AbsoluteMove raw response body: %s", response)
	return map[string]interface{}{"response": response}, nil
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
		return s.handleGetConfiguration(cmd)
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
