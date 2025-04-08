package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/use-go/onvif"
	"github.com/use-go/onvif/ptz"
	"github.com/use-go/onvif/xsd"
	onvifxsd "github.com/use-go/onvif/xsd/onvif"
)

// --- Constants for Space URIs ---
const (
	AbsolutePanTiltPositionGenericSpace     = "http://www.onvif.org/ver10/tptz/PanTiltSpaces/PositionGenericSpace"
	AbsolutePanTiltPositionSphericalDegrees = "http://www.onvif.org/ver10/tptz/PanTiltSpaces/SphericalPositionSpaceDegrees"
	AbsoluteZoomPositionGenericSpace        = "http://www.onvif.org/ver10/tptz/ZoomSpaces/PositionGenericSpace"

	RelativePanTiltTranslationGenericSpace     = "http://www.onvif.org/ver10/tptz/PanTiltSpaces/TranslationGenericSpace"
	RelativePanTiltTranslationSphericalDegrees = "http://www.onvif.org/ver10/tptz/PanTiltSpaces/SphericalTranslationSpaceDegrees"
	RelativeZoomTranslationGenericSpace        = "http://www.onvif.org/ver10/tptz/ZoomSpaces/TranslationGenericSpace"

	ContinuousPanTiltVelocityGenericSpace = "http://www.onvif.org/ver10/tptz/PanTiltSpaces/VelocityGenericSpace"
	ContinuousZoomVelocityGenericSpace    = "http://www.onvif.org/ver10/tptz/ZoomSpaces/VelocityGenericSpace"
)

// CameraConfig holds camera connection details loaded from JSON config
type CameraConfig struct {
	IP       string `json:"ip"`       // IP address with port (e.g., "192.168.1.100:80")
	Username string `json:"username"` // ONVIF username
	Password string `json:"password"` // ONVIF password
	Profile  string `json:"profile"`  // Profile token (e.g., "000" or "001")
}

// Shared flags and config
var (
	configFile string       // Path to the JSON configuration file
	config     CameraConfig // Camera configuration
	profile    string       // Profile token (populated from config)
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "onvif-ptz-cli",
	Short: "A CLI tool to control ONVIF PTZ cameras",
	Long: `onvif-ptz-cli allows you to send PTZ commands (move, zoom, stop, get status)
to network cameras supporting the ONVIF protocol.

Requires a JSON configuration file (specified with --config) containing:
- Camera IP address and port
- Username and password for authentication
- Profile token (typically "000" or "001")

Example:
  ./ptz-client --config camera.json get-status`,
	// PersistentPreRunE ensures flags are checked before any subcommand runs.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip validation for help commands and potentially others that don't need full auth/profile
		if cmd.Name() == "help" || (len(args) > 0 && args[0] == "help") || cmd.Name() == "version" {
			return nil
		}

		// Check if config file is provided
		if configFile == "" {
			return fmt.Errorf("required flag --config not set")
		}

		// Load configuration
		var err error
		config, err = loadConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Update profile for use in commands
		profile = config.Profile
		return nil
	},
}

// loadConfig loads camera configuration from a JSON file
func loadConfig(configPath string) (CameraConfig, error) {
	// If file doesn't exist but has no extension, try adding .json
	if _, err := os.Stat(configPath); os.IsNotExist(err) && filepath.Ext(configPath) == "" {
		if _, err := os.Stat(configPath + ".json"); err == nil {
			configPath = configPath + ".json"
		}
	}

	file, err := os.Open(configPath)
	if err != nil {
		return CameraConfig{}, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config CameraConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return CameraConfig{}, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate required fields
	if config.IP == "" {
		return CameraConfig{}, fmt.Errorf("missing required field 'ip' in config")
	}
	if config.Username == "" {
		return CameraConfig{}, fmt.Errorf("missing required field 'username' in config")
	}
	if config.Password == "" {
		return CameraConfig{}, fmt.Errorf("missing required field 'password' in config")
	}
	if config.Profile == "" {
		return CameraConfig{}, fmt.Errorf("missing required field 'profile' in config")
	}

	return config, nil
}

// createOnvifDevice is a helper function to create the device instance
func createOnvifDevice() (*onvif.Device, error) {
	dev, err := onvif.NewDevice(onvif.DeviceParams{
		Xaddr:    config.IP,
		Username: config.Username,
		Password: config.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ONVIF device for %s: %w", config.IP, err)
	}
	return dev, nil
}

// --- Subcommand Definitions ---

// Get Status Command
var getStatusCmd = &cobra.Command{
	Use:   "get-status",
	Short: "Get the current PTZ status (position and movement state)",
	Run: func(cmd *cobra.Command, args []string) {
		dev, err := createOnvifDevice()
		if err != nil {
			log.Fatalf("Error creating device: %v", err)
		}

		req := ptz.GetStatus{ProfileToken: onvifxsd.ReferenceToken(profile)}
		resp, err := dev.CallMethod(req)
		if err != nil {
			log.Fatalf("Failed to call GetStatus: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("Failed to read GetStatus response body: %v", err)
		}
		fmt.Printf("  Raw XML: %s\n", string(body))

		// Define the expected envelope structure
		var statusEnvelope struct {
			XMLName xml.Name `xml:"Envelope"`
			Body    struct {
				GetStatusResponse ptz.GetStatusResponse `xml:"GetStatusResponse"`
			} `xml:"Body"`
		}

		if err := xml.Unmarshal(body, &statusEnvelope); err != nil {
			log.Fatalf("Failed to unmarshal GetStatus response: %v", err)
		}

		statusResp := statusEnvelope.Body.GetStatusResponse

		fmt.Println("Current PTZ Status:")

		// Check if PTZStatus struct itself is non-zero (basic check)
		if statusResp.PTZStatus.Position.PanTilt.X == 0 && statusResp.PTZStatus.Position.PanTilt.Y == 0 && statusResp.PTZStatus.Position.Zoom.X == 0 {
			fmt.Println("Warning: PTZ values are all zero. This may indicate the camera is at its default position or no movement has occurred.")
		}

		fmt.Printf("  Pan/Tilt State: %s\n", statusResp.PTZStatus.MoveStatus.PanTilt)
		fmt.Printf("  Zoom State:     %s\n", statusResp.PTZStatus.MoveStatus.Zoom)
		pos := statusResp.PTZStatus.Position
		spaceInfo := "Normalized Generic Space"
		if pos.PanTilt.Space != "" {
			if pos.PanTilt.Space == AbsolutePanTiltPositionSphericalDegrees {
				spaceInfo = "Spherical Degrees Space"
			} else {
				spaceInfo = fmt.Sprintf("Space: %s", pos.PanTilt.Space)
			}
		}
		fmt.Printf("  Position (%s):\n", spaceInfo)
		fmt.Printf("    Pan:  %.5f\n", pos.PanTilt.X)
		fmt.Printf("    Tilt: %.5f\n", pos.PanTilt.Y)

		zoomSpaceInfo := "Normalized Generic Space"
		if pos.Zoom.Space != "" {
			zoomSpaceInfo = fmt.Sprintf("Space: %s", pos.Zoom.Space)
		}
		fmt.Printf("    Zoom: %.5f (%s, Range: 0.0 to 1.0)\n", pos.Zoom.X, zoomSpaceInfo)

		if statusResp.PTZStatus.Error != "" {
			fmt.Printf("  Error Status: %s\n", statusResp.PTZStatus.Error)
		}

		if statusResp.PTZStatus.UtcTime != "" {
			fmt.Printf("  Timestamp (UTC): %s\n", statusResp.PTZStatus.UtcTime)
		}
	},
}

// Stop Command
var (
	stopPanTilt bool
	stopZoom    bool
)
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop ongoing Pan/Tilt and/or Zoom movements",
	Run: func(cmd *cobra.Command, args []string) {
		dev, err := createOnvifDevice()
		if err != nil {
			log.Fatalf("Error creating device: %v", err)
		}

		req := ptz.Stop{
			ProfileToken: onvifxsd.ReferenceToken(profile),
			PanTilt:      xsd.Boolean(stopPanTilt),
			Zoom:         xsd.Boolean(stopZoom),
		}

		fmt.Printf("Sending Stop command (PanTilt: %v, Zoom: %v)...\n", stopPanTilt, stopZoom)
		_, err = dev.CallMethod(req)
		if err != nil {
			log.Fatalf("Failed to call Stop: %v", err)
		}
		fmt.Println("Stop command sent successfully.")
	},
}

// Continuous Move Command
var (
	panSpeed     float64
	tiltSpeed    float64
	zoomSpeed    float64
	moveDuration time.Duration
)
var continuousMoveCmd = &cobra.Command{
	Use:   "continuous-move",
	Short: "Start a continuous Pan/Tilt/Zoom movement",
	Long: `Starts moving the camera continuously at the specified speeds.
Speeds (Velocity) for Pan/Tilt/Zoom should be between -1.0 and 1.0.
Use the --duration flag to automatically stop after a period (e.g., --duration 2s).
If duration is 0 or not specified, the camera moves until a 'stop' command is sent.`,
	Run: func(cmd *cobra.Command, args []string) {
		dev, err := createOnvifDevice()
		if err != nil {
			log.Fatalf("Error creating device: %v", err)
		}

		if panSpeed < -1.0 || panSpeed > 1.0 || tiltSpeed < -1.0 || tiltSpeed > 1.0 || zoomSpeed < -1.0 || zoomSpeed > 1.0 {
			log.Fatalf("Error: Speed values (--x, --y, --z) must be between -1.0 and 1.0")
		}

		req := ptz.ContinuousMove{
			ProfileToken: onvifxsd.ReferenceToken(profile),
			Velocity: onvifxsd.PTZSpeed{
				PanTilt: onvifxsd.Vector2D{
					X: panSpeed,
					Y: tiltSpeed,
					// Space: ContinuousPanTiltVelocityGenericSpace // URI usually optional
				},
				Zoom: onvifxsd.Vector1D{
					X: zoomSpeed,
					// Space: ContinuousZoomVelocityGenericSpace // URI usually optional
				},
			},
		}

		fmt.Printf("Sending ContinuousMove (PanSpeed: %.2f, TiltSpeed: %.2f, ZoomSpeed: %.2f)...\n", panSpeed, tiltSpeed, zoomSpeed)
		_, err = dev.CallMethod(req)
		if err != nil {
			log.Fatalf("Failed to call ContinuousMove: %v", err)
		}
		fmt.Println("ContinuousMove command sent successfully.")

		if moveDuration > 0 {
			fmt.Printf("Waiting for duration: %v\n", moveDuration)
			time.Sleep(moveDuration)
			fmt.Println("Duration elapsed. Sending Stop command...")

			doStopPanTilt := panSpeed != 0 || tiltSpeed != 0
			doStopZoom := zoomSpeed != 0
			stopReq := ptz.Stop{
				ProfileToken: onvifxsd.ReferenceToken(profile),
				PanTilt:      xsd.Boolean(doStopPanTilt),
				Zoom:         xsd.Boolean(doStopZoom),
			}
			_, err = dev.CallMethod(stopReq)
			if err != nil {
				log.Fatalf("Failed to send Stop command after duration: %v", err)
			}
			fmt.Println("Stop command sent successfully after duration.")
		} else if panSpeed != 0 || tiltSpeed != 0 || zoomSpeed != 0 {
			fmt.Println("Camera is moving continuously. Use the 'stop' command to halt movement.")
		}
	},
}

// Relative Move Command
var (
	panRelative    float64
	tiltRelative   float64
	zoomRelative   float64
	speedXRelative float64
	speedYRelative float64
	speedZRelative float64
	useDegreesRel  bool // Flag variable for relative move degrees
)
var relativeMoveCmd = &cobra.Command{
	Use:   "relative-move",
	Short: "Move the camera by a relative amount",
	Long: `Moves the camera by a relative pan/tilt/zoom translation amount.
Default (without --degrees): Uses normalized generic space.
  Pan/Tilt (-x, -y): Range -1.0 to 1.0.
  Zoom (-z): Range -1.0 to 1.0.
With --degrees flag: Uses spherical degree space for Pan/Tilt.
  Pan (-x): Range -180.0 to 180.0 degrees.
  Tilt (-y): Range -90.0 to 90.0 degrees.
  Zoom (-z): Still uses normalized generic space (-1.0 to 1.0).
Optionally specify speed (normalized -1.0 to 1.0, positive usually used) for the movement.`,
	Run: func(cmd *cobra.Command, args []string) {
		dev, err := createOnvifDevice()
		if err != nil {
			log.Fatalf("Error creating device: %v", err)
		}

		// Input validation based on --degrees flag
		if useDegreesRel {
			if panRelative < -180.0 || panRelative > 180.0 {
				log.Fatalf("Error: Relative pan (-x) must be between -180.0 and 180.0 when using --degrees")
			}
			if tiltRelative < -90.0 || tiltRelative > 90.0 {
				log.Fatalf("Error: Relative tilt (-y) must be between -90.0 and 90.0 when using --degrees")
			}
		} else {
			if panRelative < -1.0 || panRelative > 1.0 {
				log.Fatalf("Error: Relative pan (-x) must be between -1.0 and 1.0 (use --degrees for degrees)")
			}
			if tiltRelative < -1.0 || tiltRelative > 1.0 {
				log.Fatalf("Error: Relative tilt (-y) must be between -1.0 and 1.0 (use --degrees for degrees)")
			}
		}
		// Zoom validation is always normalized for relative move
		if zoomRelative < -1.0 || zoomRelative > 1.0 {
			log.Fatalf("Error: Relative zoom (-z) must be between -1.0 and 1.0")
		}

		// Prepare PanTilt Vector
		panTiltVector := onvifxsd.Vector2D{
			X: panRelative,
			Y: tiltRelative,
		}
		if useDegreesRel {
			// Need to set the Space field using a pointer to the URI string
			spaceURI := RelativePanTiltTranslationSphericalDegrees
			panTiltVector.Space = xsd.AnyURI(spaceURI)
			fmt.Println("Using Spherical Degrees space for relative Pan/Tilt.")
		} else {
			// Optionally set the generic space or leave nil for default
			// spaceURI := RelativePanTiltTranslationGenericSpace
			// panTiltVector.Space = xsd.AnyURI(spaceURI)
			fmt.Println("Using Generic Normalized space for relative Pan/Tilt.")
		}

		req := ptz.RelativeMove{
			ProfileToken: onvifxsd.ReferenceToken(profile),
			Translation: onvifxsd.PTZVector{
				PanTilt: panTiltVector,
				Zoom: onvifxsd.Vector1D{
					X: zoomRelative,
					// Space: RelativeZoomTranslationGenericSpace // Zoom always generic here
				},
			},
		}

		// Add speed if specified
		if cmd.Flags().Changed("speed-x") || cmd.Flags().Changed("speed-y") || cmd.Flags().Changed("speed-z") {
			if speedXRelative < -1.0 || speedXRelative > 1.0 || speedYRelative < -1.0 || speedYRelative > 1.0 || speedZRelative < -1.0 || speedZRelative > 1.0 {
				log.Fatalf("Error: Speed values must be between -1.0 and 1.0")
			}
			req.Speed = onvifxsd.PTZSpeed{
				PanTilt: onvifxsd.Vector2D{X: speedXRelative, Y: speedYRelative},
				Zoom:    onvifxsd.Vector1D{X: speedZRelative},
			}
			fmt.Printf("Sending RelativeMove (Pan: %.3f, Tilt: %.3f, Zoom: %.3f) with Speed (X: %.2f, Y: %.2f, Z: %.2f)...\n",
				panRelative, tiltRelative, zoomRelative, speedXRelative, speedYRelative, speedZRelative)
		} else {
			fmt.Printf("Sending RelativeMove (Pan: %.3f, Tilt: %.3f, Zoom: %.3f) with default speed...\n",
				panRelative, tiltRelative, zoomRelative)
		}

		_, err = dev.CallMethod(req)
		if err != nil {
			log.Fatalf("Failed to call RelativeMove: %v", err)
		}
		fmt.Println("RelativeMove command sent successfully.")
	},
}

// Absolute Move Command
var (
	panAbsolute    float64
	tiltAbsolute   float64
	zoomAbsolute   float64
	speedXAbsolute float64
	speedYAbsolute float64
	speedZAbsolute float64
	useDegreesAbs  bool // Flag variable for absolute move degrees
)
var absoluteMoveCmd = &cobra.Command{
	Use:   "absolute-move",
	Short: "Move the camera to an absolute position",
	Long: `Moves the camera to an absolute pan/tilt/zoom position.
Default (without --degrees): Uses normalized generic space.
  Pan/Tilt position (-x, -y): Range -1.0 to 1.0.
  Zoom position (-z): Range 0.0 (out) to 1.0 (in).
With --degrees flag: Uses spherical degree space for Pan/Tilt position.
  Pan (-x): Range -180.0 to 180.0 degrees.
  Tilt (-y): Range -90.0 to 90.0 degrees.
  Zoom (-z): Still uses normalized generic space (0.0 to 1.0).
Optionally specify speed (normalized -1.0 to 1.0, positive usually used) for the movement.`,
	Run: func(cmd *cobra.Command, args []string) {
		dev, err := createOnvifDevice()
		if err != nil {
			log.Fatalf("Error creating device: %v", err)
		}

		// Input validation based on --degrees flag
		if useDegreesAbs {
			if panAbsolute < -180.0 || panAbsolute > 180.0 {
				log.Fatalf("Error: Absolute pan (-x) must be between -180.0 and 180.0 when using --degrees")
			}
			if tiltAbsolute < -90.0 || tiltAbsolute > 90.0 {
				log.Fatalf("Error: Absolute tilt (-y) must be between -90.0 and 90.0 when using --degrees")
			}
		} else {
			if panAbsolute < -1.0 || panAbsolute > 1.0 {
				log.Fatalf("Error: Absolute pan (-x) must be between -1.0 and 1.0 (use --degrees for degrees)")
			}
			if tiltAbsolute < -1.0 || tiltAbsolute > 1.0 {
				log.Fatalf("Error: Absolute tilt (-y) must be between -1.0 and 1.0 (use --degrees for degrees)")
			}
		}
		// Zoom validation is always normalized for absolute move
		if zoomAbsolute < 0.0 || zoomAbsolute > 1.0 {
			log.Fatalf("Error: Absolute zoom (-z) must be between 0.0 and 1.0")
		}

		// Prepare PanTilt Vector
		panTiltVector := onvifxsd.Vector2D{
			X: panAbsolute,
			Y: tiltAbsolute,
		}
		if useDegreesAbs {
			// Need to set the Space field using a pointer to the URI string
			spaceURI := AbsolutePanTiltPositionSphericalDegrees
			panTiltVector.Space = xsd.AnyURI(spaceURI)
			fmt.Println("Using Spherical Degrees space for absolute Pan/Tilt.")
		} else {
			// Optionally set the generic space or leave nil for default
			// spaceURI := AbsolutePanTiltPositionGenericSpace
			// panTiltVector.Space = xsd.AnyURI(spaceURI)
			fmt.Println("Using Generic Normalized space for absolute Pan/Tilt.")
		}

		req := ptz.AbsoluteMove{
			ProfileToken: onvifxsd.ReferenceToken(profile),
			Position: onvifxsd.PTZVector{
				PanTilt: panTiltVector,
				Zoom: onvifxsd.Vector1D{
					X: zoomAbsolute,
					// Space: AbsoluteZoomPositionGenericSpace // Zoom always generic here
				},
			},
		}

		// Add speed if specified
		if cmd.Flags().Changed("speed-x") || cmd.Flags().Changed("speed-y") || cmd.Flags().Changed("speed-z") {
			if speedXAbsolute < -1.0 || speedXAbsolute > 1.0 || speedYAbsolute < -1.0 || speedYAbsolute > 1.0 || speedZAbsolute < -1.0 || speedZAbsolute > 1.0 {
				log.Fatalf("Error: Speed values must be between -1.0 and 1.0")
			}
			req.Speed = onvifxsd.PTZSpeed{
				PanTilt: onvifxsd.Vector2D{X: speedXAbsolute, Y: speedYAbsolute},
				Zoom:    onvifxsd.Vector1D{X: speedZAbsolute},
			}
			fmt.Printf("Sending AbsoluteMove (Pan: %.3f, Tilt: %.3f, Zoom: %.3f) with Speed (X: %.2f, Y: %.2f, Z: %.2f)...\n",
				panAbsolute, tiltAbsolute, zoomAbsolute, speedXAbsolute, speedYAbsolute, speedZAbsolute)
		} else {
			fmt.Printf("Sending AbsoluteMove (Pan: %.3f, Tilt: %.3f, Zoom: %.3f) with default speed...\n",
				panAbsolute, tiltAbsolute, zoomAbsolute)
		}

		_, err = dev.CallMethod(req)
		if err != nil {
			log.Fatalf("Failed to call AbsoluteMove: %v\n"+
				"NOTE: Ensure values are within the camera's supported range for the chosen space (normalized or degrees).", err)
		}
		fmt.Println("AbsoluteMove command sent successfully.")
	},
}

// --- Initialization ---

func init() {
	// Add config file flag (required)
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Path to camera configuration JSON file (required)")
	rootCmd.MarkPersistentFlagRequired("config")

	// --- Add Subcommands and flags ---

	// Get Status Command
	rootCmd.AddCommand(getStatusCmd)

	// Stop Command
	stopCmd.Flags().BoolVar(&stopPanTilt, "pan-tilt", true, "Stop pan/tilt movement")
	stopCmd.Flags().BoolVar(&stopZoom, "zoom", true, "Stop zoom movement")
	rootCmd.AddCommand(stopCmd)

	// Continuous Move Command
	continuousMoveCmd.Flags().Float64VarP(&panSpeed, "x", "x", 0.0, "Pan speed/velocity (Range: -1.0 to 1.0)")
	continuousMoveCmd.Flags().Float64VarP(&tiltSpeed, "y", "y", 0.0, "Tilt speed/velocity (Range: -1.0 to 1.0)")
	continuousMoveCmd.Flags().Float64VarP(&zoomSpeed, "z", "z", 0.0, "Zoom speed/velocity (Range: -1.0 to 1.0)")
	continuousMoveCmd.Flags().DurationVarP(&moveDuration, "duration", "d", 0, "Duration to move before stopping automatically (e.g., 2s, 500ms). 0 means move continuously.")
	rootCmd.AddCommand(continuousMoveCmd)

	// Relative Move Command
	relativeMoveCmd.Flags().Float64VarP(&panRelative, "x", "x", 0.0, "Relative pan distance (normalized -1.0 to 1.0 OR degrees -180.0 to 180.0 with --degrees)")
	relativeMoveCmd.Flags().Float64VarP(&tiltRelative, "y", "y", 0.0, "Relative tilt distance (normalized -1.0 to 1.0 OR degrees -90.0 to 90.0 with --degrees)")
	relativeMoveCmd.Flags().Float64VarP(&zoomRelative, "z", "z", 0.0, "Relative zoom distance (normalized -1.0 to 1.0)")
	relativeMoveCmd.Flags().Float64Var(&speedXRelative, "speed-x", 0.5, "Speed for pan component (Range: -1.0 to 1.0)")
	relativeMoveCmd.Flags().Float64Var(&speedYRelative, "speed-y", 0.5, "Speed for tilt component (Range: -1.0 to 1.0)")
	relativeMoveCmd.Flags().Float64Var(&speedZRelative, "speed-z", 0.5, "Speed for zoom component (Range: -1.0 to 1.0)")
	relativeMoveCmd.Flags().BoolVar(&useDegreesRel, "degrees", false, "Interpret Pan/Tilt values (-x, -y) as degrees (Spherical Space)") // New Flag
	rootCmd.AddCommand(relativeMoveCmd)

	// Absolute Move Command
	absoluteMoveCmd.Flags().Float64VarP(&panAbsolute, "x", "x", 0.0, "Absolute pan position (normalized -1.0 to 1.0 OR degrees -180.0 to 180.0 with --degrees)")
	absoluteMoveCmd.Flags().Float64VarP(&tiltAbsolute, "y", "y", 0.0, "Absolute tilt position (normalized -1.0 to 1.0 OR degrees -90.0 to 90.0 with --degrees)")
	absoluteMoveCmd.Flags().Float64VarP(&zoomAbsolute, "z", "z", 0.0, "Absolute zoom position (normalized 0.0 to 1.0)")
	absoluteMoveCmd.Flags().Float64Var(&speedXAbsolute, "speed-x", 0.5, "Speed for pan component (Range: -1.0 to 1.0)")
	absoluteMoveCmd.Flags().Float64Var(&speedYAbsolute, "speed-y", 0.5, "Speed for tilt component (Range: -1.0 to 1.0)")
	absoluteMoveCmd.Flags().Float64Var(&speedZAbsolute, "speed-z", 0.5, "Speed for zoom component (Range: -1.0 to 1.0)")
	absoluteMoveCmd.Flags().BoolVar(&useDegreesAbs, "degrees", false, "Interpret Pan/Tilt position (-x, -y) as degrees (Spherical Space)") // New Flag
	_ = absoluteMoveCmd.MarkFlagRequired("x")
	_ = absoluteMoveCmd.MarkFlagRequired("y")
	_ = absoluteMoveCmd.MarkFlagRequired("z")
	rootCmd.AddCommand(absoluteMoveCmd)
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func main() {
	Execute()
}
