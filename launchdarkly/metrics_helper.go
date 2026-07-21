package launchdarkly

import (
	"fmt"
	"strings"
)

func metricIdToKeys(id string) (projectKey string, flagKey string, err error) {
	if strings.Count(id, "/") != 1 {
		return "", "", fmt.Errorf("found unexpected metric id format: %q expected format: 'project_key/metric_key'", id)
	}
	parts := strings.SplitN(id, "/", 2)
	projectKey, flagKey = parts[0], parts[1]
	return projectKey, flagKey, nil
}
