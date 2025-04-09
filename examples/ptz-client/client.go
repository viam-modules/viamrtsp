package main

import (
	"bytes"
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
	"github.com/use-go/onvif/media"
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

// --- Custom Structs for GetStatus Response ---
type CustomGetStatusEnvelope struct {
	XMLName xml.Name            `xml:"http://www.w3.org/2003/05/soap-envelope Envelope"`
	Body    CustomGetStatusBody `xml:"http://www.w3.org/2003/05/soap-envelope Body"`
}

type CustomGetStatusBody struct {
	GetResponse CustomGetStatusResponse `xml:"http://www.onvif.org/ver20/ptz/wsdl GetStatusResponse"`
}

type CustomGetStatusResponse struct {
	PTZStatus CustomPTZStatus `xml:"http://www.onvif.org/ver20/ptz/wsdl PTZStatus"`
}

type CustomPTZStatus struct {
	Position   CustomPosition   `xml:"http://www.onvif.org/ver10/schema Position"`
	MoveStatus CustomMoveStatus `xml:"http://www.onvif.org/ver10/schema MoveStatus"`
	UtcTime    string           `xml:"http://www.onvif.org/ver10/schema UtcTime"`
}

type CustomPosition struct {
	PanTilt CustomVector2D `xml:"http://www.onvif.org/ver10/schema PanTilt"`
	Zoom    CustomVector1D `xml:"http://www.onvif.org/ver10/schema Zoom"`
}

type CustomVector2D struct {
	X     float64 `xml:"x,attr"`
	Y     float64 `xml:"y,attr"`
	Space string  `xml:"space,attr,omitempty"`
}

type CustomVector1D struct {
	X     float64 `xml:"x,attr"`
	Space string  `xml:"space,attr,omitempty"`
}

type CustomMoveStatus struct {
	PanTilt string `xml:"http://www.onvif.org/ver10/schema PanTilt"`
	Zoom    string `xml:"http://www.onvif.org/ver10/schema Zoom"`
}

// CameraConfig holds camera connection details loaded from JSON config
type CameraConfig struct {
	IP       string `json:"ip"`       // IP address with port (e.g., "192.168.1.100:80")
	Username string `json:"username"` // ONVIF username
	Password string `json:"password"` // ONVIF password
}

// Shared flags and config
var (
	configFilePath string
	config         CameraConfig
	profile        string // ONVIF media profile token
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ptz-client",
	Short: "A CLI tool to control ONVIF PTZ cameras",
	Long: `ptz-client allows you to send PTZ commands (move, zoom, stop, get status)
to network cameras supporting the ONVIF protocol.

Requires a JSON configuration file (specified with --config) containing:
- Camera IP address and port
- Username and password for authentication

And a profile token (specified with --profile) which is typically "000" or "001".

Example:
  ./ptz-client --config camera.json --profile 001 get-status`,
	// PersistentPreRunE ensures shared flags are checked before any subcommand runs.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip validation for help/version commands
		isHelp := cmd.Name() == "help" || (len(args) > 0 && args[0] == "help")
		if isHelp || cmd.Name() == "version" {
			return nil
		}

		if configFilePath == "" {
			return fmt.Errorf("required flag --config not set for command '%s'", cmd.Name())
		}
		var err error
		config, err = loadConfig(configFilePath)
		if err != nil {
			return fmt.Errorf("failed to load configuration using '%s' for command '%s': %w", configFilePath, cmd.Name(), err)
		}

		// Conditionally require --profile
		if cmd.Name() != "get-profiles" { // get-profiles doesn't need a profile
			if profile == "" {
				return fmt.Errorf("required flag --profile not set for command '%s'", cmd.Name())
			}
		}

		return nil
	},
	// Disable default Cobra commands
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

// loadConfig loads camera configuration from a JSON file
func loadConfig(configPath string) (CameraConfig, error) {
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

	// Validate
	if config.IP == "" {
		return CameraConfig{}, fmt.Errorf("missing required field 'ip' in config")
	}
	if config.Username == "" {
		return CameraConfig{}, fmt.Errorf("missing required field 'username' in config")
	}
	if config.Password == "" {
		return CameraConfig{}, fmt.Errorf("missing required field 'password' in config")
	}

	return config, nil
}

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

// --- Cobra Subcommands ---

var getProfilesCmd = &cobra.Command{
	Use:   "get-profiles",
	Short: "Retrieve available media profiles",
	Long:  `This command helps you discover which media profile tokens the camera supports for actuation.`,
	Run: func(cmd *cobra.Command, args []string) {
		dev, err := createOnvifDevice()
		if err != nil {
			log.Fatalf("Failed to create ONVIF device: %v", err)
		}
		fmt.Println("Retrieving media profiles...")
		profilesReq := media.GetProfiles{}
		profilesRes, err := dev.CallMethod(profilesReq)
		if err != nil {
			fmt.Printf("Warning: Failed to get media profiles: %v\n", err)
			return
		}
		body, _ := io.ReadAll(profilesRes.Body)
		defer profilesRes.Body.Close()
		fmt.Printf("Raw Media Profiles Response:\n%s\n\n", string(body))

		// Parse profiles from XML
		var profilesResponse struct {
			XMLName xml.Name `xml:"Envelope"`
			Body    struct {
				GetProfilesResponse struct {
					Profiles []struct {
						Token string `xml:"token,attr"`
					} `xml:"Profiles"`
				} `xml:"GetProfilesResponse"`
			} `xml:"Body"`
		}

		if err := xml.NewDecoder(bytes.NewReader(body)).Decode(&profilesResponse); err == nil {
			fmt.Println("Parsed Media Profiles:")
			for _, profile := range profilesResponse.Body.GetProfilesResponse.Profiles {
				fmt.Printf(" - %s\n", profile.Token)
			}
		} else {
			fmt.Printf("Failed to decode media profiles: %v\n", err)
		}
	},
}

var getStatusCmd = &cobra.Command{
	Use:   "get-status",
	Short: "Retrieve current PTZ status",
	Long: `Retrieve and display the current Pan/Tilt/Zoom position,
movement status, and UTC time from the camera for the specified profile.`,
	Run: func(cmd *cobra.Command, args []string) {
		dev, err := createOnvifDevice()
		if err != nil {
			log.Fatalf("Failed to create ONVIF device: %v", err)
		}

		req := ptz.GetStatus{ProfileToken: onvifxsd.ReferenceToken(profile)}
		fmt.Printf("Sending GetStatus request for profile: %s\n", profile)

		res, err := dev.CallMethod(req)
		if err != nil {
			log.Fatalf("Failed to call GetStatus: %v", err)
		}
		defer res.Body.Close()

		bodyBytes, err := io.ReadAll(res.Body)
		if err != nil {
			log.Fatalf("Failed to read response body: %v", err)
		}
		fmt.Printf("Raw XML Response:\n%s\n", string(bodyBytes))

		var statusEnvelope CustomGetStatusEnvelope
		err = xml.Unmarshal(bodyBytes, &statusEnvelope)
		if err != nil {
			log.Fatalf("Failed to unmarshal GetStatus response using custom structs: %v\nRaw XML was:\n%s", err, string(bodyBytes))
		}

		ptzStatus := statusEnvelope.Body.GetResponse.PTZStatus
		fmt.Println("\n--- PTZ Status ---")
		fmt.Printf("  Position:\n")
		fmt.Printf("    Pan/Tilt: X=%.6f, Y=%.6f (Space: %s)\n",
			ptzStatus.Position.PanTilt.X,
			ptzStatus.Position.PanTilt.Y,
			ptzStatus.Position.PanTilt.Space)
		fmt.Printf("    Zoom:     X=%.6f (Space: %s)\n",
			ptzStatus.Position.Zoom.X,
			ptzStatus.Position.Zoom.Space)
		fmt.Printf("  Move Status:\n")
		fmt.Printf("    Pan/Tilt: %s\n", ptzStatus.MoveStatus.PanTilt)
		fmt.Printf("    Zoom:     %s\n", ptzStatus.MoveStatus.Zoom)
		fmt.Printf("  UTC Time:   %s\n", ptzStatus.UtcTime)
		fmt.Println("------------------")
	},
}

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
				},
				Zoom: onvifxsd.Vector1D{
					X: zoomSpeed,
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
			// Default to Generic Normalized space for relative Pan/Tilt
			fmt.Println("Using Generic Normalized space for relative Pan/Tilt.")
		}

		req := ptz.RelativeMove{
			ProfileToken: onvifxsd.ReferenceToken(profile),
			Translation: onvifxsd.PTZVector{
				PanTilt: panTiltVector,
				Zoom: onvifxsd.Vector1D{
					X: zoomRelative, // zoom always generic space
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
			// Default to Generic Normalized space for absolute Pan/Tilt
			fmt.Println("Using Generic Normalized space for absolute Pan/Tilt.")
		}

		req := ptz.AbsoluteMove{
			ProfileToken: onvifxsd.ReferenceToken(profile),
			Position: onvifxsd.PTZVector{
				PanTilt: panTiltVector,
				Zoom: onvifxsd.Vector1D{
					X: zoomAbsolute, // zoom always generic space
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
			log.Fatalf("Failed to call AbsoluteMove: %v\n", err)
		}
		fmt.Println("AbsoluteMove command sent successfully.")
	},
}

// --- Initialization ---

func init() {
	// Add config file flag (required)
	rootCmd.PersistentFlags().StringVarP(&configFilePath, "config", "c", "", "Path to camera configuration JSON file (required)")
	rootCmd.MarkPersistentFlagRequired("config")

	// Add profile flag
	rootCmd.PersistentFlags().StringVarP(&profile, "profile", "p", "", "Profile token (typically \"000\" or \"001\")")

	// Get Profiles Command
	rootCmd.AddCommand(getProfilesCmd)

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

	// Disable default commands
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func main() {
	Execute()
}
