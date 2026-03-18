package launchdarkly

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// validateJsonStringFunc returns a ValidateFunc that validates a string is valid JSON.
func validateJsonStringFunc(v interface{}, k string) (warns []string, errs []error) {
	s, ok := v.(string)
	if !ok {
		return nil, []error{fmt.Errorf("%q: expected type string", k)}
	}
	if s == "" {
		return nil, nil
	}
	var js interface{}
	if err := json.Unmarshal([]byte(s), &js); err != nil {
		return nil, []error{fmt.Errorf("%q: invalid JSON: %s", k, err)}
	}
	return nil, nil
}

// suppressEquivalentJsonDiffs suppresses diffs caused by semantically equivalent JSON values
// (e.g., different key ordering).
func suppressEquivalentJsonDiffs(k, old, new string, d *schema.ResourceData) bool {
	if old == "" && new == "" {
		return true
	}
	if old == "" || new == "" {
		return false
	}
	var oldJSON, newJSON interface{}
	if err := json.Unmarshal([]byte(old), &oldJSON); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(new), &newJSON); err != nil {
		return false
	}
	return reflect.DeepEqual(oldJSON, newJSON)
}

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
