package launchdarkly

import "fmt"

// stripEmptyMapValues removes keys whose values are empty maps or empty strings
// from a top-level map. This prevents API-added defaults (e.g. "custom":{}) from
// creating spurious diffs between the user's config and the API response.
func stripEmptyMapValues(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		switch val := v.(type) {
		case map[string]interface{}:
			if len(val) > 0 {
				result[k] = val
			}
		case string:
			if val != "" {
				result[k] = val
			}
		default:
			if v != nil {
				result[k] = v
			}
		}
	}
	return result
}

// isEmptyModelMap returns true if the model map only contains empty/zero values
// (as returned by the API default: {"custom":{},"modelName":"","parameters":{}}).
func isEmptyModelMap(m map[string]interface{}) bool {
	for _, v := range m {
		switch val := v.(type) {
		case string:
			if val != "" {
				return false
			}
		case map[string]interface{}:
			if len(val) > 0 {
				return false
			}
		default:
			if v != nil {
				return false
			}
		}
	}
	return true
}

func variationIdToKeys(id string) (projectKey, configKey, variationKey string, err error) {
	parts := splitID(id, 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("import ID must be in the format project_key/config_key/variation_key, got: %q", id)
	}
	return parts[0], parts[1], parts[2], nil
}
