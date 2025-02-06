package launchdarkly

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go/v17"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testAccDataSourceFeatureFlagEnvironment = `
data "launchdarkly_feature_flag_environment" "test" {
	env_key = "%s"
	flag_id = "%s"
}
`
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
	_, err := testAccFeatureFlagScaffold(client, projectKey, flagBody)
	if err != nil {
		return nil, err
	}

	// patch feature flag with env-specific config
	patch := ldapi.NewPatchWithComment(envConfigPatches)
	patch.SetComment("Terraform feature flag env data source test")

	_, _, err = client.ld.FeatureFlagsApi.PatchFeatureFlag(client.ctx, projectKey, flagKey).PatchWithComment(*patch).Execute()

	if err != nil {
		// delete project if anything fails because otherwise we will see a
		// 409 error later and have to clean it up manually
		_ = testAccProjectScaffoldDelete(client, projectKey)
		return nil, fmt.Errorf("failed to create feature flag env config: %s", err.Error())
	}
	flag, _, err := client.ld.FeatureFlagsApi.GetFeatureFlag(client.ctx, projectKey, flagKey).Execute()

	if err != nil {
		_ = testAccProjectScaffoldDelete(client, projectKey)
		return nil, fmt.Errorf("failed to get feature flag: %s", err.Error())
	}

	return flag, nil
}

func TestAccDataSourceFeatureFlagEnvironment_noMatchReturnsError(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	envKey := "bad-env"
	flagKey := "flag-no-env"
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S)
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

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	envKey := "test"
	flagKey := "test-env-config"
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S)
	require.NoError(t, err)

	rules := []rule{
		{
			Description: strPtr("test rule"),
			Variation:   intPtr(1),
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
		patchReplace(basePatchPath+"offVariation", 1),
		patchReplace(basePatchPath+"targets", targets),
		patchReplace(basePatchPath+"fallthrough", fall),
	}
	flag, err := testAccDataSourceFeatureFlagEnvironmentScaffold(client, projectKey, envKey, flagKey, patches)
	require.NoError(t, err)

	defer func() {
		err := testAccProjectScaffoldDelete(client, projectKey)
		require.NoError(t, err)
	}()

	thisConfig := flag.Environments[envKey]
	otherConfig := flag.Environments["production"]

	flagId := projectKey + "/" + flagKey
	resourceName := "data.launchdarkly_feature_flag_environment.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceFeatureFlagEnvironment, envKey, flagId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, FLAG_ID),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, envKey),
					resource.TestCheckResourceAttr(resourceName, ON, fmt.Sprint(thisConfig.On)),
					resource.TestCheckResourceAttr(resourceName, TRACK_EVENTS, fmt.Sprint(thisConfig.TrackEvents)),
					resource.TestCheckResourceAttr(resourceName, "rules.0.description", *thisConfig.Rules[0].Description),
					resource.TestCheckResourceAttr(resourceName, "rules.0.variation", fmt.Sprint(*thisConfig.Rules[0].Variation)),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.attribute", thisConfig.Rules[0].Clauses[0].Attribute),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.op", thisConfig.Rules[0].Clauses[0].Op),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.0", fmt.Sprint(thisConfig.Rules[0].Clauses[0].Values[0])),
					resource.TestCheckResourceAttr(resourceName, "prerequisites.0.flag_key", thisConfig.Prerequisites[0].Key),
					resource.TestCheckResourceAttr(resourceName, "prerequisites.0.variation", fmt.Sprint(thisConfig.Prerequisites[0].Variation)),
					resource.TestCheckResourceAttr(resourceName, OFF_VARIATION, fmt.Sprint(*thisConfig.OffVariation)),
					resource.TestCheckResourceAttr(resourceName, "targets.0.values.#", fmt.Sprint(len(thisConfig.Targets[0].Values))),
					resource.TestCheckResourceAttr(resourceName, "targets.0.variation", "1"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceFeatureFlagEnvironment, "production", flagId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, FLAG_ID),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "production"),
					resource.TestCheckResourceAttr(resourceName, ON, fmt.Sprint(otherConfig.On)),
					resource.TestCheckResourceAttr(resourceName, TRACK_EVENTS, fmt.Sprint(otherConfig.TrackEvents)),
					resource.TestCheckResourceAttr(resourceName, "rules.#", fmt.Sprint(len(otherConfig.Rules))),
					resource.TestCheckResourceAttr(resourceName, "prerequisites.#", fmt.Sprint(len(otherConfig.Prerequisites))),
					resource.TestCheckResourceAttr(resourceName, "targets.#", fmt.Sprint(len(otherConfig.Targets))),
				),
			},
		},
	})
}

func TestAccDataSourceFeatureFlagEnvironment_WithContextFields(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	envKey := "test"
	flagKey := "test-env-config"
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S)
	require.NoError(t, err)

	testContextKind := "test-kind"
	rules := []rule{
		{
			Variation: intPtr(1),
			Clauses: []ldapi.Clause{
				{
					Attribute:   "thing",
					Op:          "contains",
					Values:      []interface{}{"test"},
					ContextKind: &testContextKind,
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
	contextTargets := []ldapi.Target{
		{
			Values:      []string{"test1", "test2"},
			Variation:   1,
			ContextKind: &testContextKind,
		},
	}
	email := "email"
	rollout := ldapi.Rollout{
		Variations: []ldapi.WeightedVariation{
			{Variation: int32(0), Weight: int32(20000)},
			{Variation: int32(1), Weight: int32(80000)},
		},
		BucketBy: &email,
		// allow ContextKind to default
	}
	fall := fallthroughModel{
		Rollout: &rollout,
	}

	basePatchPath := "/environments/" + envKey + "/"
	patches := []ldapi.PatchOperation{
		patchReplace(basePatchPath+"on", true),
		patchReplace(basePatchPath+"trackEvents", true),
		patchReplace(basePatchPath+"rules", rules),
		patchReplace(basePatchPath+"prerequisites", prerequisites),
		patchReplace(basePatchPath+"offVariation", 1),
		patchReplace(basePatchPath+"targets", targets),
		patchReplace(basePatchPath+"contextTargets", contextTargets),
		patchReplace(basePatchPath+"fallthrough", fall),
	}
	flag, err := testAccDataSourceFeatureFlagEnvironmentScaffold(client, projectKey, envKey, flagKey, patches)
	require.NoError(t, err)

	defer func() {
		err := testAccProjectScaffoldDelete(client, projectKey)
		require.NoError(t, err)
	}()

	thisConfig := flag.Environments[envKey]
	otherConfig := flag.Environments["production"]

	flagId := projectKey + "/" + flagKey
	resourceName := "data.launchdarkly_feature_flag_environment.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceFeatureFlagEnvironment, envKey, flagId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, FLAG_ID),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, envKey),
					resource.TestCheckResourceAttr(resourceName, ON, fmt.Sprint(thisConfig.On)),
					resource.TestCheckResourceAttr(resourceName, TRACK_EVENTS, fmt.Sprint(thisConfig.TrackEvents)),
					resource.TestCheckResourceAttr(resourceName, "rules.0.variation", fmt.Sprint(*thisConfig.Rules[0].Variation)),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.attribute", thisConfig.Rules[0].Clauses[0].Attribute),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.op", thisConfig.Rules[0].Clauses[0].Op),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.0", fmt.Sprint(thisConfig.Rules[0].Clauses[0].Values[0])),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.context_kind", *thisConfig.Rules[0].Clauses[0].ContextKind),
					resource.TestCheckResourceAttr(resourceName, "prerequisites.0.flag_key", thisConfig.Prerequisites[0].Key),
					resource.TestCheckResourceAttr(resourceName, "prerequisites.0.variation", fmt.Sprint(thisConfig.Prerequisites[0].Variation)),
					resource.TestCheckResourceAttr(resourceName, OFF_VARIATION, fmt.Sprint(*thisConfig.OffVariation)),
					resource.TestCheckResourceAttr(resourceName, "fallthrough.0.context_kind", "user"), // set by default
					resource.TestCheckResourceAttr(resourceName, "targets.0.values.#", fmt.Sprint(len(thisConfig.Targets[0].Values))),
					resource.TestCheckResourceAttr(resourceName, "targets.0.variation", "1"),
					resource.TestCheckResourceAttr(resourceName, "context_targets.0.values.#", fmt.Sprint(len(thisConfig.Targets[0].Values))),
					resource.TestCheckResourceAttr(resourceName, "context_targets.0.values.0", thisConfig.ContextTargets[0].Values[0]),
					resource.TestCheckResourceAttr(resourceName, "context_targets.0.values.1", thisConfig.ContextTargets[0].Values[1]),
					resource.TestCheckResourceAttr(resourceName, "context_targets.0.variation", "1"),
					resource.TestCheckResourceAttr(resourceName, "context_targets.0.context_kind", *thisConfig.ContextTargets[0].ContextKind),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceFeatureFlagEnvironment, "production", flagId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, FLAG_ID),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "production"),
					resource.TestCheckResourceAttr(resourceName, ON, fmt.Sprint(otherConfig.On)),
					resource.TestCheckResourceAttr(resourceName, TRACK_EVENTS, fmt.Sprint(otherConfig.TrackEvents)),
					resource.TestCheckResourceAttr(resourceName, "rules.#", fmt.Sprint(len(otherConfig.Rules))),
					resource.TestCheckResourceAttr(resourceName, "prerequisites.#", fmt.Sprint(len(otherConfig.Prerequisites))),
					resource.TestCheckResourceAttr(resourceName, "targets.#", fmt.Sprint(len(otherConfig.Targets))),
				),
			},
		},
	})
}
