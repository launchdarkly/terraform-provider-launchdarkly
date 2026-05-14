package launchdarkly

import (
	"net/http"

	ldapi "github.com/launchdarkly/api-client-go/v22"
)

// rule is the patch payload shape used by acceptance tests that patch
// FFE rules via the LD API directly. Kept here so test files don't need
// to inline it; the framework resource builds its own ffeRulePayload
// (which is the same shape).
type rule struct {
	Description *string        `json:"description,omitempty"`
	Variation   *int           `json:"variation,omitempty"`
	Rollout     *ldapi.Rollout `json:"rollout,omitempty"`
	Clauses     []ldapi.Clause `json:"clauses,omitempty"`
}

// fallthroughModel is the patch payload shape for the FFE fallthrough
// block. Shared with the acceptance test that patches a flag's
// fallthrough directly via the LD API.
type fallthroughModel struct {
	Variation *int           `json:"variation,omitempty"`
	Rollout   *ldapi.Rollout `json:"rollout,omitempty"`
}

// getFeatureFlagEnvironment uses the LD API's `env` query param to read
// a single environment's view of a feature flag in one round-trip.
// Shared between data_source_feature_flag_environment_framework.go and
// resource_feature_flag_environment_framework.go.
func getFeatureFlagEnvironment(client *Client, projectKey, flagKey, environmentKey string) (*ldapi.FeatureFlag, *http.Response, error) {
	var flag *ldapi.FeatureFlag
	var res *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		flag, res, err = client.ld.FeatureFlagsApi.GetFeatureFlag(client.ctx, projectKey, flagKey).Env(environmentKey).Execute()
		return err
	})
	return flag, res, err
}
