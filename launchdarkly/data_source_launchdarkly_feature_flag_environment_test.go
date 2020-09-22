package launchdarkly

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	ldapi "github.com/launchdarkly/api-client-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testAccDataSourceFeatureFlagEnvironment = `
data_source "launchdarkly_feature_flag" "test" {
	project_key = %s
	env_key = %s
	flag_id = %s
}`
)

func testAccDataSourceFeatureFlagEnvironmentScaffold(client *Client, projectKey, envKey, flagKey string, envConfigPatches []ldapi.PatchOperation) (*ldapi.FeatureFlag, error) {
	// create a flag
	flagBody := ldapi.FeatureFlagBody{
		Name: "Feature Flag Env Data Source Test",
		Key:  flagKey,
		Variations: []ldapi.Variation{
			{Value: intfPtr(true)},
			{Value: intfPtr(false)},
		},
	}
	_, err := testAccDataSourceFeatureFlagScaffold(client, projectKey, flagBody)
	if err != nil {
		return nil, err
	}

	// patch feature flag with env-specific config
	patch := ldapi.PatchComment{
		Comment: "Terraform feature flag env data source test",
		Patch:   envConfigPatches,
	}
	_, _, err = handleRateLimit(func() (interface{}, *http.Response, error) {
		return handleNoConflict(func() (interface{}, *http.Response, error) {
			return client.ld.FeatureFlagsApi.PatchFeatureFlag(client.ctx, projectKey, flagKey, patch)
		})
	})
	if err != nil {
		// delete project if anything fails because otherwise we will see a
		// 409 error later and have to clean it up manually
		_ = testAccDataSourceProjectDelete(client, projectKey)
		return nil, fmt.Errorf("failed to create feature flag env config: %s", err.Error())
	}
	flagRaw, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.FeatureFlagsApi.GetFeatureFlag(client.ctx, projectKey, flagKey, nil)
	})
	if err != nil {
		_ = testAccDataSourceProjectDelete(client, projectKey)
		return nil, fmt.Errorf("failed to get feature flag: %s", err.Error())
	}

	flag, ok := flagRaw.(ldapi.FeatureFlag)
	if !ok {
		_ = testAccDataSourceProjectDelete(client, projectKey)
		return nil, fmt.Errorf("failed to create feature flag env config")
	}
	return &flag, nil
}

func TestAccDataSourceFeatureFlagEnvironment_noMatchReturnsError(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := "ff-env-ds-test"
	envKey := "bad-env"
	flagKey := "flag-no-env"
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false)
	require.NoError(t, err)

	// create some fake config
	patches := []ldapi.PatchOperation{
		patchReplace("/environments/"+envKey+"/on", false),
	}
	_, err = testAccDataSourceFeatureFlagEnvironmentScaffold(client, projectKey, envKey, flagKey, patches)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "400 Bad Request")
}

func TestAccDataSourceFeatureFlagEnvironment_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := "ff-env-ds-test2"
	envKey := "test"
	flagKey := "test-env-config"
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false)
	require.NoError(t, err)

	rules := []rule{
		{
			Variation: intPtr(1),
			Clauses: []ldapi.Clause{
				{
					Attribute: "thing",
					Op:        "contains",
					Values:    []interface{}{"test"},
				},
			},
		},
	}
	prerequisites := []ldapi.Prerequisite{
		{
			Key:       "some-other-flag",
			Variation: 1,
		},
	}
	targets := []ldapi.Target{
		{
			Values:    []string{"some@email.com", "some_other@email.com"},
			Variation: 1,
		},
	}
	fall := fallthroughModel{
		Variation: intPtr(0),
	}

	basePatchPath := "/environments/" + envKey + "/"
	patches := []ldapi.PatchOperation{
		patchReplace(basePatchPath+"on", true),
		patchReplace(basePatchPath+"trackEvents", true),
		patchReplace(basePatchPath+"rules", rules),
		patchReplace(basePatchPath+"prerequisites", prerequisites),
		patchReplace(basePatchPath+"offVariation", 0),
		patchReplace(basePatchPath+"targets", targets),
		patchReplace(basePatchPath+"fallthrough", fall),
	}
	flag, err := testAccDataSourceFeatureFlagEnvironmentScaffold(client, projectKey, envKey, flagKey, patches)
	require.NoError(t, err)

	thisConfig := flag.Environments[envKey]
	otherConfig := flag.Environments["production"]
	// ensure the config is different from the original default that should still apply to the other env
	assert.Equal(t, thisConfig.On, true)
	assert.Equal(t, thisConfig.TrackEvents, true)
	assert.Equal(t, thisConfig.Rules[0].Variation, int32(1))
	assert.Equal(t, thisConfig.Rules[0].Clauses[0].Attribute, "thing")
	assert.Equal(t, thisConfig.Rules[0].Clauses[0].Op, "contains")
	assert.Equal(t, thisConfig.Rules[0].Clauses[0].Values[0], "test")
	assert.Equal(t, thisConfig.Prerequisites[0].Key, "some-other-flag")
	assert.Equal(t, thisConfig.Prerequisites[0].Variation, int32(1))
	assert.Equal(t, thisConfig.OffVariation, int32(0))
	assert.Equal(t, thisConfig.Targets[0].Variation, int32(1))
	assert.Len(t, thisConfig.Targets[0].Values, 2)
	assert.Equal(t, thisConfig.Fallthrough_.Variation, int32(0))
	assert.NotEqual(t, thisConfig.On, otherConfig.On)
	assert.NotEqual(t, thisConfig.TrackEvents, otherConfig.TrackEvents)
	assert.Len(t, otherConfig.Rules, 0)
	assert.Len(t, otherConfig.Prerequisites, 0)
	assert.Len(t, otherConfig.Targets, 0)

	err = testAccDataSourceProjectDelete(client, projectKey)
	require.NoError(t, err)
}
