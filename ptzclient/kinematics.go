package ptzclient

import "encoding/json"

// SVA joint limits in degrees used when movements.absolute is not configured
// or when the ONVIF space is PositionGenericSpace (normalized -1.0 to 1.0).
const (
	defaultPanMinDeg  = -180.0
	defaultPanMaxDeg  = 180.0
	defaultTiltMinDeg = -90.0
	defaultTiltMaxDeg = 90.0
)

// svaModel is the JSON structure for a Viam SVA kinematics file.
// Translations are in millimeters; joint min/max are in degrees.
type svaModel struct {
	Name               string    `json:"name"`
	KinematicParamType string    `json:"kinematic_param_type"`
	Links              []svaLink `json:"links"`
	Joints             []svaJoint `json:"joints"`
}

type svaLink struct {
	ID          string       `json:"id"`
	Parent      string       `json:"parent"`
	Translation svaTranslation `json:"translation"`
}

type svaTranslation struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type svaJoint struct {
	ID     string     `json:"id"`
	Type   string     `json:"type"`
	Parent string     `json:"parent"`
	Axis   svaTranslation `json:"axis"`
	Min    float64    `json:"min"`
	Max    float64    `json:"max"`
}

// buildSVA generates the SVA kinematics model for the pan-tilt gimbal.
// Joint limits are derived from the absolute position config if present;
// otherwise conservative defaults are used.
func buildSVA(cfg *Config) ([]byte, error) {
	panMin, panMax, tiltMin, tiltMax := svaJointLimits(cfg)

	model := svaModel{
		Name:               "onvif_pan_tilt",
		KinematicParamType: "SVA",
		Links: []svaLink{
			{
				ID:          "base_link",
				Parent:      "world",
				Translation: svaTranslation{X: 0, Y: 0, Z: 0},
			},
			{
				ID:          "pan_link",
				Parent:      "pan_joint",
				Translation: svaTranslation{X: 0, Y: 0, Z: 0},
			},
			{
				// 1mm offset along X — the gaze end effector point.
				ID:          "gaze",
				Parent:      "tilt_joint",
				Translation: svaTranslation{X: 1, Y: 0, Z: 0},
			},
		},
		Joints: []svaJoint{
			{
				ID:     "pan_joint",
				Type:   "revolute",
				Parent: "base_link",
				Axis:   svaTranslation{X: 0, Y: 0, Z: 1}, // vertical axis
				Min:    panMin,
				Max:    panMax,
			},
			{
				ID:     "tilt_joint",
				Type:   "revolute",
				Parent: "pan_link",
				Axis:   svaTranslation{X: 0, Y: 1, Z: 0}, // horizontal axis
				Min:    tiltMin,
				Max:    tiltMax,
			},
		},
	}

	return json.Marshal(model)
}

// svaJointLimits returns pan and tilt joint limits in degrees derived from
// the absolute position config. If the config uses PositionGenericSpace
// (normalized -1.0 to 1.0), the ONVIF range is scaled to degrees using the
// defaults. If it uses SphericalPositionSpaceDegrees, the values are used
// directly. If movements.absolute is not configured, defaults are returned.
func svaJointLimits(cfg *Config) (panMin, panMax, tiltMin, tiltMax float64) {
	abs, ok := cfg.Movements["absolute"]
	if !ok {
		return defaultPanMinDeg, defaultPanMaxDeg, defaultTiltMinDeg, defaultTiltMaxDeg
	}

	pt := abs.PanTilt
	if pt.Space == AbsolutePanTiltPositionSphericalDegrees {
		// Values are already in degrees.
		return pt.XMin, pt.XMax, pt.YMin, pt.YMax
	}

	// PositionGenericSpace or unknown: treat as normalized -1.0 to 1.0
	// and scale to the default degree range.
	panMin = onvifToDegrees(pt.XMin, -1, 1, defaultPanMinDeg, defaultPanMaxDeg)
	panMax = onvifToDegrees(pt.XMax, -1, 1, defaultPanMinDeg, defaultPanMaxDeg)
	tiltMin = onvifToDegrees(pt.YMin, -1, 1, defaultTiltMinDeg, defaultTiltMaxDeg)
	tiltMax = onvifToDegrees(pt.YMax, -1, 1, defaultTiltMinDeg, defaultTiltMaxDeg)
	return panMin, panMax, tiltMin, tiltMax
}
