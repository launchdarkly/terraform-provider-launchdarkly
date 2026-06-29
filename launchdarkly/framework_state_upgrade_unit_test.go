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
