package launchdarkly

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	ldapi "github.com/launchdarkly/api-client-go/v25"
	"github.com/stretchr/testify/require"
)

func TestViewAssociationSettingNeedsPatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		plan     types.Bool
		state    types.Bool
		isCreate bool
		want     bool
	}{
		{
			name:     "create default false does not patch",
			plan:     types.BoolValue(false),
			isCreate: true,
			want:     false,
		},
		{
			name:     "create unset does not patch",
			plan:     types.BoolNull(),
			isCreate: true,
			want:     false,
		},
		{
			name:     "create true patches",
			plan:     types.BoolValue(true),
			isCreate: true,
			want:     true,
		},
		{
			name:  "update unchanged false does not patch",
			plan:  types.BoolValue(false),
			state: types.BoolValue(false),
			want:  false,
		},
		{
			name:  "update unchanged true does not patch",
			plan:  types.BoolValue(true),
			state: types.BoolValue(true),
			want:  false,
		},
		{
			name:  "update false to true patches",
			plan:  types.BoolValue(true),
			state: types.BoolValue(false),
			want:  true,
		},
		{
			name:  "update true to false patches",
			plan:  types.BoolValue(false),
			state: types.BoolValue(true),
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, viewAssociationSettingNeedsPatch(tt.plan, tt.state, tt.isCreate))
		})
	}
}

// testEnvPlanObject builds a fully-populated nested-environment plan object
// as the framework would hand it to Create (every Optional+Computed bool
// resolved to a concrete value by its default or the config).
func testEnvPlanObject(t *testing.T, key string, tags []string, approvals types.Object) types.Object {
	t.Helper()
	tagVals := make([]attr.Value, 0, len(tags))
	for _, tag := range tags {
		tagVals = append(tagVals, types.StringValue(tag))
	}
	tagSet := types.SetNull(types.StringType)
	if len(tags) > 0 {
		tagSet = types.SetValueMust(types.StringType, tagVals)
	}
	return types.ObjectValueMust(environmentAttrTypes, map[string]attr.Value{
		KEY:                  types.StringValue(key),
		NAME:                 types.StringValue("Production"),
		COLOR:                types.StringValue("ff0000"),
		CRITICAL:             types.BoolValue(true),
		API_KEY:              types.StringUnknown(),
		MOBILE_KEY:           types.StringUnknown(),
		CLIENT_SIDE_ID:       types.StringUnknown(),
		DEFAULT_TTL:          types.Int64Value(15),
		SECURE_MODE:          types.BoolValue(true),
		DEFAULT_TRACK_EVENTS: types.BoolValue(true),
		REQUIRE_COMMENTS:     types.BoolValue(true),
		CONFIRM_CHANGES:      types.BoolValue(true),
		TAGS:                 tagSet,
		APPROVAL_SETTINGS:    approvals,
	})
}

func testApprovalSettingsObject() types.Object {
	return types.ObjectValueMust(frameworkApprovalSettingsObjectAttrTypes, map[string]attr.Value{
		REQUIRED:                    types.BoolValue(true),
		CAN_REVIEW_OWN_REQUEST:      types.BoolValue(false),
		MIN_NUM_APPROVALS:           types.Int64Value(2),
		CAN_APPLY_DECLINED_CHANGES:  types.BoolValue(true),
		REQUIRED_APPROVAL_TAGS:      types.ListNull(types.StringType),
		SERVICE_KIND:                types.StringValue("launchdarkly"),
		SERVICE_CONFIG:              types.MapNull(types.StringType),
		AUTO_APPLY_APPROVED_CHANGES: types.BoolValue(false),
	})
}

func patchPaths(ops []ldapi.PatchOperation) []string {
	paths := make([]string, 0, len(ops))
	for _, op := range ops {
		paths = append(paths, op.Path)
	}
	return paths
}

// TestProjectPostFromPlan_CarriesCreateTimeAttributes pins the fix for
// tag-scoped project creation: tags, default_client_side_availability and
// the environments' full shape must be on the POST body, because
// LaunchDarkly authorizes POST /projects against the key and tags in the
// request and a tag-scoped role denies an untagged create.
func TestProjectPostFromPlan_CarriesCreateTimeAttributes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	plan := ProjectResourceModel{
		Key:  types.StringValue("team-a-proj"),
		Name: types.StringValue("Team A"),
		Tags: types.SetValueMust(types.StringType, []attr.Value{
			types.StringValue("FG-FeatureManagement"),
			types.StringValue("terraform"),
		}),
		DefaultClientSideAvailability: types.ObjectValueMust(projectCSAAttrTypes, map[string]attr.Value{
			USING_ENVIRONMENT_ID: types.BoolValue(true),
			USING_MOBILE_KEY:     types.BoolValue(false),
		}),
		Environments: types.MapValueMust(environmentObjectType, map[string]attr.Value{
			"production": testEnvPlanObject(t, "production", []string{"prod"}, types.ObjectNull(frameworkApprovalSettingsObjectAttrTypes)),
		}),
	}

	post, diags := projectPostFromPlan(ctx, plan)
	require.False(t, diags.HasError(), diags)

	require.Equal(t, "team-a-proj", post.Key)
	require.Equal(t, "Team A", post.Name)
	require.ElementsMatch(t, []string{"FG-FeatureManagement", "terraform"}, post.Tags)
	require.NotNil(t, post.DefaultClientSideAvailability)
	require.True(t, post.DefaultClientSideAvailability.UsingEnvironmentId)
	require.False(t, post.DefaultClientSideAvailability.UsingMobileKey)

	require.Len(t, post.Environments, 1)
	env := post.Environments[0]
	require.Equal(t, "production", env.Key)
	require.Equal(t, "Production", env.Name)
	require.Equal(t, "ff0000", env.Color)
	require.Equal(t, []string{"prod"}, env.Tags)
	require.NotNil(t, env.DefaultTtl)
	require.EqualValues(t, 15, *env.DefaultTtl)
	for name, got := range map[string]*bool{
		"secureMode":         env.SecureMode,
		"defaultTrackEvents": env.DefaultTrackEvents,
		"requireComments":    env.RequireComments,
		"confirmChanges":     env.ConfirmChanges,
		"critical":           env.Critical,
	} {
		require.NotNil(t, got, name)
		require.True(t, *got, name)
	}
}

// TestProjectPostFromPlan_OmitsUnsetOptionals: attributes the user did not
// configure stay off the POST body so the API applies its own defaults.
func TestProjectPostFromPlan_OmitsUnsetOptionals(t *testing.T) {
	t.Parallel()

	plan := ProjectResourceModel{
		Key:                           types.StringValue("bare"),
		Name:                          types.StringValue("Bare"),
		Tags:                          types.SetNull(types.StringType),
		DefaultClientSideAvailability: types.ObjectNull(projectCSAAttrTypes),
		Environments:                  types.MapNull(environmentObjectType),
	}

	post, diags := projectPostFromPlan(context.Background(), plan)
	require.False(t, diags.HasError(), diags)
	require.Nil(t, post.Tags)
	require.Nil(t, post.DefaultClientSideAvailability)
	require.Nil(t, post.Environments)
}

// TestEnvironmentPatchFromModels_FreshlyPosted: an environment that was just
// POSTed (with the project, or on its own) carries its full shape already,
// so the follow-up PATCH is limited to approval_settings — and is empty when
// none are configured. An environment adopted from LaunchDarkly (not freshly
// posted) still gets the full reconciliation patch.
func TestEnvironmentPatchFromModels_FreshlyPosted(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var noApprovals environmentModel
	require.False(t, testEnvPlanObject(t, "prod", []string{"prod"}, types.ObjectNull(frameworkApprovalSettingsObjectAttrTypes)).As(ctx, &noApprovals, basetypes.ObjectAsOptions{}).HasError())
	var withApprovals environmentModel
	require.False(t, testEnvPlanObject(t, "prod", []string{"prod"}, testApprovalSettingsObject()).As(ctx, &withApprovals, basetypes.ObjectAsOptions{}).HasError())

	t.Run("freshly posted without approvals patches nothing", func(t *testing.T) {
		t.Parallel()
		ops, diags := environmentPatchFromModels(ctx, environmentModel{}, false, noApprovals, true)
		require.False(t, diags.HasError(), diags)
		require.Empty(t, ops)
	})

	t.Run("freshly posted with approvals patches only approvalSettings", func(t *testing.T) {
		t.Parallel()
		ops, diags := environmentPatchFromModels(ctx, environmentModel{}, false, withApprovals, true)
		require.False(t, diags.HasError(), diags)
		require.NotEmpty(t, ops)
		for _, p := range patchPaths(ops) {
			require.True(t, strings.HasPrefix(p, "/approvalSettings/"), "unexpected non-approval patch %q on a freshly posted environment", p)
		}
	})

	t.Run("freshly posted ignores stale prior approval state", func(t *testing.T) {
		t.Parallel()
		// State remembers approvals on an env that was deleted out of band and
		// has just been recreated: no removal patch must be emitted against
		// the brand-new environment.
		ops, diags := environmentPatchFromModels(ctx, withApprovals, true, noApprovals, true)
		require.False(t, diags.HasError(), diags)
		require.Empty(t, ops)
	})

	t.Run("adopted environment gets the full reconciliation patch", func(t *testing.T) {
		t.Parallel()
		ops, diags := environmentPatchFromModels(ctx, environmentModel{}, false, noApprovals, false)
		require.False(t, diags.HasError(), diags)
		require.Subset(t, patchPaths(ops), []string{
			"/name", "/color", "/defaultTtl", "/secureMode", "/defaultTrackEvents",
			"/requireComments", "/confirmChanges", "/critical", "/tags",
		})
	})
}
