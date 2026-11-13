package launchdarkly

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironmentDefaultReleasePlanIdToKeys(t *testing.T) {
	projectKey, envKey, err := environmentDefaultReleasePlanIdToKeys("proj/production")
	require.NoError(t, err)
	assert.Equal(t, "proj", projectKey)
	assert.Equal(t, "production", envKey)

	for _, bad := range []string{"", "proj", "proj/", "/env", "proj/env/extra"} {
		_, _, err := environmentDefaultReleasePlanIdToKeys(bad)
		assert.Error(t, err, bad)
	}
}

func TestEnvironmentDefaultReleasePlanEndpoint(t *testing.T) {
	assert.Equal(t,
		"https://app.launchdarkly.com/api/v2/projects/proj/environments/production/default-release-plan",
		environmentDefaultReleasePlanEndpoint("", "proj", "production"))
	assert.Equal(t,
		"http://localhost:3000/api/v2/projects/proj/environments/production/default-release-plan",
		environmentDefaultReleasePlanEndpoint("http://localhost:3000", "proj", "production"))
	assert.Equal(t,
		"https://app.ld.catamorphic.com/api/v2/projects/proj/environments/production/default-release-plan",
		environmentDefaultReleasePlanEndpoint("app.ld.catamorphic.com", "proj", "production"))
}
