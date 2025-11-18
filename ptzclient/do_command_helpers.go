package ptzclient

import (
	"fmt"
)

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
	return string(durationVal)
}
