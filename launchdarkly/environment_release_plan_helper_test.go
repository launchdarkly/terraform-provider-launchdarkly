package launchdarkly

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironmentReleasePlanIdToKeys(t *testing.T) {
	projectKey, envKey, key, err := environmentReleasePlanIdToKeys("proj/production/checkout-rollout")
	require.NoError(t, err)
	assert.Equal(t, "proj", projectKey)
	assert.Equal(t, "production", envKey)
	assert.Equal(t, "checkout-rollout", key)

	for _, bad := range []string{"", "proj", "proj/env", "proj//key", "proj/env/key/extra"} {
		_, _, _, err := environmentReleasePlanIdToKeys(bad)
		assert.Error(t, err, bad)
	}
}

func TestEnvironmentReleasePlanEndpoint(t *testing.T) {
	assert.Equal(t,
		"https://app.launchdarkly.com/api/v2/projects/proj/environments/production/release-plans",
		environmentReleasePlanEndpoint("", "proj", "production", ""))
	assert.Equal(t,
		"https://app.launchdarkly.com/api/v2/projects/proj/environments/production/release-plans/checkout-rollout",
		environmentReleasePlanEndpoint("", "proj", "production", "checkout-rollout"))
	assert.Equal(t,
		"http://localhost:3000/api/v2/projects/proj/environments/production/release-plans/plan",
		environmentReleasePlanEndpoint("http://localhost:3000", "proj", "production", "plan"))
	assert.Equal(t,
		"https://app.ld.catamorphic.com/api/v2/projects/proj/environments/production/release-plans/plan",
		environmentReleasePlanEndpoint("app.ld.catamorphic.com", "proj", "production", "plan"))
}
