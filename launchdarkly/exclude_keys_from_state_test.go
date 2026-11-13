package launchdarkly

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	ldapi "github.com/launchdarkly/api-client-go/v24"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExcludedKeyValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, types.StringValue("sdk-123"), excludedKeyValue(types.BoolNull(), "sdk-123"), "null keeps the key")
	assert.Equal(t, types.StringValue("sdk-123"), excludedKeyValue(types.BoolUnknown(), "sdk-123"), "unknown keeps the key")
	assert.Equal(t, types.StringValue("sdk-123"), excludedKeyValue(types.BoolValue(false), "sdk-123"), "false keeps the key")
	assert.Equal(t, types.StringNull(), excludedKeyValue(types.BoolValue(true), "sdk-123"), "true stores null")
}

func testEnvObject(t *testing.T, key string, exclude types.Bool, api, mobile, csid attr.Value) attr.Value {
	t.Helper()
	return types.ObjectValueMust(environmentAttrTypes, map[string]attr.Value{
		KEY:                     types.StringValue(key),
		NAME:                    types.StringValue(key),
		COLOR:                   types.StringValue("000000"),
		CRITICAL:                types.BoolValue(false),
		API_KEY:                 api,
		MOBILE_KEY:              mobile,
		CLIENT_SIDE_ID:          csid,
		DEFAULT_TTL:             types.Int64Value(0),
		SECURE_MODE:             types.BoolValue(false),
		DEFAULT_TRACK_EVENTS:    types.BoolValue(false),
		REQUIRE_COMMENTS:        types.BoolValue(false),
		CONFIRM_CHANGES:         types.BoolValue(false),
		TAGS:                    types.SetNull(types.StringType),
		APPROVAL_SETTINGS:       types.ObjectNull(frameworkApprovalSettingsObjectAttrTypes),
		EXCLUDE_KEYS_FROM_STATE: exclude,
	})
}

func TestMarkEnvSecretsUnknownExcludeKeysFromState(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	empty := types.StringValue("")

	stateMap := types.MapValueMust(environmentObjectType, map[string]attr.Value{
		// Keys stored in state (opt-in off).
		"stored": testEnvObject(t, "stored", types.BoolNull(), types.StringValue("sdk-stored"), types.StringValue("mob-stored"), types.StringValue("csid-stored")),
		// Keys excluded in state (opt-in on).
		"excluded": testEnvObject(t, "excluded", types.BoolValue(true), types.StringNull(), types.StringNull(), types.StringValue("csid-excluded")),
	})

	planMap := types.MapValueMust(environmentObjectType, map[string]attr.Value{
		// Unchanged opt-out: reuse state.
		"stored": testEnvObject(t, "stored", types.BoolNull(), empty, empty, empty),
		// Opt back out: state is null, so the keys are unknown until apply.
		"excluded": testEnvObject(t, "excluded", types.BoolValue(false), empty, empty, empty),
		// New env opting in: keys are planned as null.
		"new-excluded": testEnvObject(t, "new-excluded", types.BoolValue(true), empty, empty, empty),
		// New env opting out: keys are unknown.
		"new-stored": testEnvObject(t, "new-stored", types.BoolNull(), empty, empty, empty),
	})

	out, diags := markEnvSecretsUnknown(ctx, planMap, stateMap)
	require.False(t, diags.HasError(), "%v", diags)

	attrsFor := func(key string) map[string]attr.Value {
		obj, ok := out.Elements()[key].(basetypes.ObjectValue)
		require.True(t, ok, "missing env %q", key)
		return obj.Attributes()
	}

	stored := attrsFor("stored")
	assert.Equal(t, types.StringValue("sdk-stored"), stored[API_KEY])
	assert.Equal(t, types.StringValue("mob-stored"), stored[MOBILE_KEY])
	assert.Equal(t, types.StringValue("csid-stored"), stored[CLIENT_SIDE_ID])

	excluded := attrsFor("excluded")
	assert.True(t, excluded[API_KEY].IsUnknown())
	assert.True(t, excluded[MOBILE_KEY].IsUnknown())
	assert.Equal(t, types.StringValue("csid-excluded"), excluded[CLIENT_SIDE_ID])

	newExcluded := attrsFor("new-excluded")
	assert.True(t, newExcluded[API_KEY].IsNull())
	assert.True(t, newExcluded[MOBILE_KEY].IsNull())
	assert.True(t, newExcluded[CLIENT_SIDE_ID].IsUnknown())

	newStored := attrsFor("new-stored")
	assert.True(t, newStored[API_KEY].IsUnknown())
	assert.True(t, newStored[MOBILE_KEY].IsUnknown())
	assert.True(t, newStored[CLIENT_SIDE_ID].IsUnknown())

	// An env switching from stored to excluded plans null.
	planMap = types.MapValueMust(environmentObjectType, map[string]attr.Value{
		"stored": testEnvObject(t, "stored", types.BoolValue(true), empty, empty, empty),
	})
	out, diags = markEnvSecretsUnknown(ctx, planMap, stateMap)
	require.False(t, diags.HasError(), "%v", diags)
	stored = attrsFor("stored")
	assert.True(t, stored[API_KEY].IsNull())
	assert.True(t, stored[MOBILE_KEY].IsNull())
	assert.Equal(t, types.StringValue("csid-stored"), stored[CLIENT_SIDE_ID])
}

func TestEnvironmentObjectFromAPIExcludeKeysFromState(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := ldapi.Environment{
		Key:       "production",
		Name:      "Production",
		Color:     "000000",
		ApiKey:    "sdk-123",
		MobileKey: "mob-123",
		Id:        "csid-123",
		Tags:      []string{},
	}

	decode := func(obj basetypes.ObjectValue) environmentModel {
		var m environmentModel
		require.False(t, obj.As(ctx, &m, basetypes.ObjectAsOptions{}).HasError())
		return m
	}

	// No prior (import): keys stored, flag null.
	obj, diags := environmentObjectFromAPI(ctx, env, nil)
	require.False(t, diags.HasError(), "%v", diags)
	m := decode(obj)
	assert.Equal(t, "sdk-123", m.APIKey.ValueString())
	assert.Equal(t, "mob-123", m.MobileKey.ValueString())
	assert.True(t, m.ExcludeKeysFromState.IsNull())

	// Prior opted in: keys null, flag preserved, client-side ID kept.
	prior := environmentModel{
		Tags:                 types.SetNull(types.StringType),
		ApprovalSettings:     types.ObjectNull(frameworkApprovalSettingsObjectAttrTypes),
		ExcludeKeysFromState: types.BoolValue(true),
	}
	obj, diags = environmentObjectFromAPI(ctx, env, &prior)
	require.False(t, diags.HasError(), "%v", diags)
	m = decode(obj)
	assert.True(t, m.APIKey.IsNull())
	assert.True(t, m.MobileKey.IsNull())
	assert.Equal(t, "csid-123", m.ClientSideID.ValueString())
	assert.True(t, m.ExcludeKeysFromState.ValueBool())

	// Prior explicitly opted out: keys stored, flag preserved.
	prior.ExcludeKeysFromState = types.BoolValue(false)
	obj, diags = environmentObjectFromAPI(ctx, env, &prior)
	require.False(t, diags.HasError(), "%v", diags)
	m = decode(obj)
	assert.Equal(t, "sdk-123", m.APIKey.ValueString())
	assert.False(t, m.ExcludeKeysFromState.ValueBool())
	assert.False(t, m.ExcludeKeysFromState.IsNull())
}
