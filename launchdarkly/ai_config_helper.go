package launchdarkly

import "fmt"

func aiConfigIdToKeys(id string) (projectKey, configKey string, err error) {
	parts := splitID(id, 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("import ID must be in the format project_key/config_key, got: %q", id)
	}
	return parts[0], parts[1], nil
}
