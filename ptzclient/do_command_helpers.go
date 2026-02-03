package ptzclient

import (
	"fmt"
)

// --- Speed Helpers ---

// speeds holds extracted speed parameters.
type speeds struct {
	pan, tilt, zoom float64
	provided        bool
}

// extractSpeeds extracts optional speed parameters from a command map.
// Returns speeds.provided=false if no speed parameters were given.
func extractSpeeds(cmd map[string]interface{}) speeds {
	_, p := cmd["pan_speed"]
	_, t := cmd["tilt_speed"]
	_, z := cmd["zoom_speed"]
	if !p && !t && !z {
		return speeds{}
	}
	return speeds{
		pan:      getOptionalFloat64(cmd, "pan_speed", defaultPanSpeed),
		tilt:     getOptionalFloat64(cmd, "tilt_speed", defaultTiltSpeed),
		zoom:     getOptionalFloat64(cmd, "zoom_speed", defaultZoomSpeed),
		provided: true,
	}
}

// validateSpeeds checks that speed values are within valid range.
// If allowNegative is true, range is [-1.0, 1.0], otherwise [0.0, 1.0].
func validateSpeeds(p, t, z float64, allowNegative bool) error {
	lower := 0.0
	if allowNegative {
		lower = -1.0
	}
	if p < lower || p > 1.0 || t < lower || t > 1.0 || z < lower || z > 1.0 {
		return fmt.Errorf("speed values must be between %.1f and 1.0", lower)
	}
	return nil
}

// --- Argument Parsing Helpers ---

// getString extracts a string argument, returning an error if missing or wrong type.
func getString(cmd map[string]interface{}, key string) (string, error) {
	val, ok := cmd[key]
	if !ok {
		return "", fmt.Errorf("missing required argument: %s", key)
	}
	strVal, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("argument '%s' must be a string, got %T", key, val)
	}
	return strVal, nil
}

// getFloat64 extracts a float64 argument, returning an error if missing or wrong type.
func getFloat64(cmd map[string]interface{}, key string) (float64, error) {
	val, ok := cmd[key]
	if !ok {
		return 0, fmt.Errorf("missing required argument: %s", key)
	}
	floatVal, ok := val.(float64)
	if !ok {
		// Also allow int types and cast them
		intVal, isInt := val.(int)
		if isInt {
			return float64(intVal), nil
		}
		return 0, fmt.Errorf("argument '%s' must be a number (float64), got %T", key, val)
	}
	return floatVal, nil
}

// getOptionalFloat64 extracts an optional float64 argument.
func getOptionalFloat64(cmd map[string]interface{}, key string, defaultVal float64) float64 {
	val, ok := cmd[key]
	if !ok {
		return defaultVal
	}
	floatVal, ok := val.(float64)
	if !ok {
		intVal, isInt := val.(int)
		if isInt {
			return float64(intVal)
		}
		return defaultVal
	}
	return floatVal
}

// getOptionalBool extracts an optional boolean argument.
func getOptionalBool(cmd map[string]interface{}, key string, defaultVal bool) bool {
	val, ok := cmd[key]
	if !ok {
		return defaultVal
	}
	boolVal, ok := val.(bool)
	if !ok {
		return defaultVal
	}
	return boolVal
}

// getOptionalString extracts an optional string argument.
func getOptionalString(cmd map[string]interface{}, key string) (string, bool) {
	val, ok := cmd[key]
	if !ok {
		return "", false
	}
	strVal, ok := val.(string)
	if !ok {
		return "", false
	}
	return strVal, true
}

// getOptionalDuration extracts an optional duration argument.
func getOptionalDuration(cmd map[string]interface{}, key string, defaultVal string) string {
	val, ok := cmd[key]
	if !ok {
		return defaultVal
	}
	durationVal, ok := val.(string)
	if !ok {
		return defaultVal
	}
	return durationVal
}
