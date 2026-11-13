package launchdarkly

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

// ffeTestBaselineModel returns a minimal, valid FeatureFlagEnvironmentResourceModel
// with all collection attributes null and a concrete fallthrough, so
// buildFFEPatches can run without error. Individual tests override
// OffVariation.
func ffeTestBaselineModel(t *testing.T) FeatureFlagEnvironmentResourceModel {
	t.Helper()
	fallthrough_, d := types.ObjectValue(ffeFallthroughAttrTypes, map[string]attr.Value{
		VARIATION:       types.Int64Value(0),
		BUCKET_BY:       types.StringNull(),
		CONTEXT_KIND:    types.StringNull(),
		ROLLOUT_WEIGHTS: types.ListNull(types.Int64Type),
	})
	require.False(t, d.HasError(), "failed to build baseline fallthrough object: %v", d)
	return FeatureFlagEnvironmentResourceModel{
		On:             types.BoolValue(false),
		TrackEvents:    types.BoolValue(false),
		Rules:          types.ListNull(types.ObjectType{AttrTypes: ffeRuleAttrTypes}),
		Prerequisites:  types.ListNull(types.ObjectType{AttrTypes: ffePrerequisiteAttrTypes}),
		Targets:        types.SetNull(types.ObjectType{AttrTypes: ffeTargetAttrTypes}),
		ContextTargets: types.SetNull(types.ObjectType{AttrTypes: ffeTargetAttrTypes}),
		Fallthrough:    fallthrough_,
	}
}

// TestBuildFFEPatches_OffVariation locks in the issue #482 behavior, in
// particular that a null off_variation only produces a JSON Patch remove when
// the live environment actually has an offVariation — LaunchDarkly returns 400
// invalid_patch on a remove of an absent path (Cursor Bugbot finding on PR
// #483).
func TestBuildFFEPatches_OffVariation(t *testing.T) {
	const envKey = "test"
	offPath := ffePatchPath(envKey, "offVariation")

	// find returns the op of the offVariation patch, or "" if absent.
	find := func(t *testing.T, plan, state FeatureFlagEnvironmentResourceModel, isCreate, live bool) string {
		t.Helper()
		patches, d := buildFFEPatches(context.Background(), envKey, plan, state, isCreate, live)
		require.False(t, d.HasError(), "buildFFEPatches errored: %v", d)
		for _, p := range patches {
			if p.Path == offPath {
				return p.Op
			}
		}
		return ""
	}

	t.Run("create, value set -> replace", func(t *testing.T) {
		plan := ffeTestBaselineModel(t)
		plan.OffVariation = types.Int64Value(1)
		require.Equal(t, "replace", find(t, plan, FeatureFlagEnvironmentResourceModel{}, true, false))
	})

	t.Run("create, null, live default present -> remove", func(t *testing.T) {
		plan := ffeTestBaselineModel(t)
		plan.OffVariation = types.Int64Null()
		require.Equal(t, "remove", find(t, plan, FeatureFlagEnvironmentResourceModel{}, true, true))
	})

	t.Run("create, null, env already Not set -> no patch", func(t *testing.T) {
		plan := ffeTestBaselineModel(t)
		plan.OffVariation = types.Int64Null()
		// live=false: a remove here would 400 invalid_patch.
		require.Equal(t, "", find(t, plan, FeatureFlagEnvironmentResourceModel{}, true, false))
	})

	t.Run("update, set -> null with prior value -> remove", func(t *testing.T) {
		plan := ffeTestBaselineModel(t)
		plan.OffVariation = types.Int64Null()
		state := ffeTestBaselineModel(t)
		state.OffVariation = types.Int64Value(2)
		require.Equal(t, "remove", find(t, plan, state, false, true))
	})

	t.Run("update, null -> null (already unset) -> no patch", func(t *testing.T) {
		plan := ffeTestBaselineModel(t)
		plan.OffVariation = types.Int64Null()
		state := ffeTestBaselineModel(t)
		state.OffVariation = types.Int64Null()
		require.Equal(t, "", find(t, plan, state, false, false))
	})

	t.Run("update, null -> 0 sets a valid, distinct value -> replace", func(t *testing.T) {
		plan := ffeTestBaselineModel(t)
		plan.OffVariation = types.Int64Value(0)
		state := ffeTestBaselineModel(t)
		state.OffVariation = types.Int64Null()
		require.Equal(t, "replace", find(t, plan, state, false, false))
	})
}
