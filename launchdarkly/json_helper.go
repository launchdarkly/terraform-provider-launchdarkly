package launchdarkly

import (
	"encoding/json"
	"fmt"
)

// jsonStringToMap converts a JSON-encoded string to a map[string]interface{}.
// Returns nil map and nil error for an empty string.
func jsonStringToMap(jsonStr string) (map[string]interface{}, error) {
	if jsonStr == "" {
		return nil, nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %s", err)
	}
	return result, nil
}

// mapToJsonString converts a map[string]interface{} to a JSON-encoded string.
// Returns an empty string for a nil map.
func mapToJsonString(m map[string]interface{}) (string, error) {
	if m == nil {
		return "", nil
	}
	bytes, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("error serializing to JSON: %s", err)
	}
	return string(bytes), nil
}
