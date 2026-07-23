package launchdarkly

import (
	"fmt"
)

func modelConfigIdToKeys(id string) (projectKey, modelConfigKey string, err error) {
	parts := splitID(id, 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("import ID must be in the format project_key/model_config_key, got: %q", id)
	}
	return parts[0], parts[1], nil
}
