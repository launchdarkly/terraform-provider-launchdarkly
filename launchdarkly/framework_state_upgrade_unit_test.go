package launchdarkly

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// These tests cover the v0 (SDKv2 block) → v3 (single object) projection
// helpers used by the feature_flag / project / feature_flag_environment
// state upgraders (REL-14237). The v2 → v3.0.0 GA upgrade path is the only
// state-compat boundary that matters, so the list→object conversion it
// performs is exercised directly here (no LaunchDarkly API / token needed).

func csaV0List(t *testing.T, attrTypes map[string]attr.Type, env, mobile bool) types.List {
	t.Helper()
	obj := types.ObjectValueMust(attrTypes, map[string]attr.Value{
		USING_ENVIRONMENT_ID: types.BoolValue(env),
		USING_MOBILE_KEY:     types.BoolValue(mobile),
	})
	return types.ListValueMust(types.ObjectType{AttrTypes: attrTypes}, []attr.Value{obj})
}

func TestCSAObjectFromV0List(t *testing.T) {
	ctx := context.Background()

	t.Run("feature_flag populated list projects to object", func(t *testing.T) {
		obj, diags := csaObjectFromV0List(ctx, csaV0List(t, featureFlagCSAAttrTypes, true, false), featureFlagCSAAttrTypes)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if obj.IsNull() {
			t.Fatal("expected populated object")
		}
		var m struct {
			UsingEnvironmentID types.Bool `tfsdk:"using_environment_id"`
			UsingMobileKey     types.Bool `tfsdk:"using_mobile_key"`
		}
		obj.As(ctx, &m, basetypes.ObjectAsOptions{})
		if !m.UsingEnvironmentID.ValueBool() || m.UsingMobileKey.ValueBool() {
			t.Errorf("values not preserved: env=%v mobile=%v", m.UsingEnvironmentID, m.UsingMobileKey)
		}
	})

	t.Run("project attr types also supported", func(t *testing.T) {
		obj, diags := csaObjectFromV0List(ctx, csaV0List(t, projectCSAAttrTypes, false, true), projectCSAAttrTypes)
		if diags.HasError() || obj.IsNull() {
			t.Fatalf("expected populated object, diags=%v null=%v", diags, obj.IsNull())
		}
	})

	t.Run("null/empty list projects to null object", func(t *testing.T) {
		null := types.ListNull(types.ObjectType{AttrTypes: featureFlagCSAAttrTypes})
		obj, _ := csaObjectFromV0List(ctx, null, featureFlagCSAAttrTypes)
		if !obj.IsNull() {
			t.Error("null list must project to null object")
		}
		empty := types.ListValueMust(types.ObjectType{AttrTypes: featureFlagCSAAttrTypes}, []attr.Value{})
		obj, _ = csaObjectFromV0List(ctx, empty, featureFlagCSAAttrTypes)
		if !obj.IsNull() {
			t.Error("empty list must project to null object")
		}
	})
}

func TestDefaultsObjectFromV0List(t *testing.T) {
	ctx := context.Background()
	objType := types.ObjectType{AttrTypes: featureFlagDefaultsAttrTypes}

	t.Run("populated", func(t *testing.T) {
		el := types.ObjectValueMust(featureFlagDefaultsAttrTypes, map[string]attr.Value{
			ON_VARIATION:  types.Int64Value(0),
			OFF_VARIATION: types.Int64Value(2),
		})
		obj, diags := defaultsObjectFromV0List(ctx, types.ListValueMust(objType, []attr.Value{el}))
		if diags.HasError() || obj.IsNull() {
			t.Fatalf("expected object, diags=%v null=%v", diags, obj.IsNull())
		}
		var m struct {
			OnVariation  types.Int64 `tfsdk:"on_variation"`
			OffVariation types.Int64 `tfsdk:"off_variation"`
		}
		obj.As(ctx, &m, basetypes.ObjectAsOptions{})
		if m.OnVariation.ValueInt64() != 0 || m.OffVariation.ValueInt64() != 2 {
			t.Errorf("values not preserved: on=%d off=%d", m.OnVariation.ValueInt64(), m.OffVariation.ValueInt64())
		}
	})

	t.Run("null", func(t *testing.T) {
		obj, _ := defaultsObjectFromV0List(ctx, types.ListNull(objType))
		if !obj.IsNull() {
			t.Error("null list must project to null object")
		}
	})
}

func TestEnvironmentsMapFromV0List(t *testing.T) {
	ctx := context.Background()

	// v0 env element = current environmentAttrTypes plus the inline KEY.
	v0Attr := map[string]attr.Type{KEY: types.StringType}
	for k, v := range environmentAttrTypes {
		v0Attr[k] = v
	}
	v0ObjType := types.ObjectType{AttrTypes: v0Attr}
	approvalObjType := types.ObjectType{AttrTypes: frameworkApprovalSettingsObjectAttrTypes}

	approval := func(required bool, min int64) basetypes.ListValue {
		return types.ListValueMust(approvalObjType, []attr.Value{
			types.ObjectValueMust(frameworkApprovalSettingsObjectAttrTypes, map[string]attr.Value{
				REQUIRED:                    types.BoolValue(required),
				CAN_REVIEW_OWN_REQUEST:      types.BoolValue(false),
				MIN_NUM_APPROVALS:           types.Int64Value(min),
				CAN_APPLY_DECLINED_CHANGES:  types.BoolValue(true),
				REQUIRED_APPROVAL_TAGS:      types.ListValueMust(types.StringType, []attr.Value{}),
				SERVICE_KIND:                types.StringValue("launchdarkly"),
				SERVICE_CONFIG:              types.MapValueMust(types.StringType, map[string]attr.Value{}),
				AUTO_APPLY_APPROVED_CHANGES: types.BoolValue(false),
			}),
		})
	}

	mkEnv := func(name string, approvals attr.Value) func(key string) attr.Value {
		return func(key string) attr.Value {
			return types.ObjectValueMust(v0Attr, map[string]attr.Value{
				KEY:                  types.StringValue(key),
				NAME:                 types.StringValue(name),
				COLOR:                types.StringValue("000000"),
				CRITICAL:             types.BoolValue(false),
				API_KEY:              types.StringValue(""),
				MOBILE_KEY:           types.StringValue(""),
				CLIENT_SIDE_ID:       types.StringValue(""),
				DEFAULT_TTL:          types.Int64Value(0),
				SECURE_MODE:          types.BoolValue(false),
				DEFAULT_TRACK_EVENTS: types.BoolValue(false),
				REQUIRE_COMMENTS:     types.BoolValue(false),
				CONFIRM_CHANGES:      types.BoolValue(false),
				TAGS:                 types.SetValueMust(types.StringType, []attr.Value{}),
				APPROVAL_SETTINGS:    approvals,
			})
		}
	}

	list := types.ListValueMust(v0ObjType, []attr.Value{
		mkEnv("Production", approval(true, 2))("production"),       // real approval → preserved
		mkEnv("Test", approval(false, 1))("test"),                  // matches API defaults → nulled
		mkEnv("Staging", types.ListNull(approvalObjType))("stage"), // null → stays null
	})

	m, diags := environmentsMapFromV0List(ctx, list)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	models := map[string]environmentModel{}
	if d := m.ElementsAs(ctx, &models, false); d.HasError() {
		t.Fatalf("decode map: %v", d)
	}
	if len(models) != 3 {
		t.Fatalf("expected 3 envs keyed by env key, got %d: %v", len(models), models)
	}
	if got := models["production"].Name.ValueString(); got != "Production" {
		t.Errorf("production name not preserved: %q", got)
	}
	if got := models["production"].Key.ValueString(); got != "production" {
		t.Errorf("production key not preserved: %q", got)
	}
	if models["production"].ApprovalSettings.IsNull() || len(models["production"].ApprovalSettings.Elements()) != 1 {
		t.Error("real approval_settings must be preserved on production")
	}
	if !models["test"].ApprovalSettings.IsNull() {
		t.Error("API-default approval_settings must be nulled on test")
	}
	if !models["stage"].ApprovalSettings.IsNull() {
		t.Error("null approval_settings must stay null on stage")
	}

	// A null/empty v0 list must project to an EMPTY (not null) map so the next
	// Read manages none rather than importing every environment.
	if nm, _ := environmentsMapFromV0List(ctx, types.ListNull(v0ObjType)); nm.IsNull() || len(nm.Elements()) != 0 {
		t.Errorf("null list must project to an empty (non-null) map, got null=%v len=%d", nm.IsNull(), len(nm.Elements()))
	}
}

func TestFFEFallthroughObjectFromV0List(t *testing.T) {
	ctx := context.Background()
	objType := types.ObjectType{AttrTypes: ffeFallthroughAttrTypes}

	el := types.ObjectValueMust(ffeFallthroughAttrTypes, map[string]attr.Value{
		VARIATION:       types.Int64Value(1),
		BUCKET_BY:       types.StringNull(),
		CONTEXT_KIND:    types.StringValue("user"),
		ROLLOUT_WEIGHTS: types.ListNull(types.Int64Type),
	})
	obj, diags := ffeFallthroughObjectFromV0List(ctx, types.ListValueMust(objType, []attr.Value{el}))
	if diags.HasError() || obj.IsNull() {
		t.Fatalf("expected object, diags=%v null=%v", diags, obj.IsNull())
	}
	var m struct {
		Variation      types.Int64  `tfsdk:"variation"`
		BucketBy       types.String `tfsdk:"bucket_by"`
		ContextKind    types.String `tfsdk:"context_kind"`
		RolloutWeights types.List   `tfsdk:"rollout_weights"`
	}
	obj.As(ctx, &m, basetypes.ObjectAsOptions{})
	if m.Variation.ValueInt64() != 1 || m.ContextKind.ValueString() != "user" {
		t.Errorf("values not preserved: variation=%d context_kind=%q", m.Variation.ValueInt64(), m.ContextKind.ValueString())
	}

	if obj, _ := ffeFallthroughObjectFromV0List(ctx, types.ListNull(objType)); !obj.IsNull() {
		t.Error("null list must project to null object")
	}
}
