package launchdarkly

// framework_state_upgrade.go holds helpers used by per-resource
// schema-version 0 → 1 state upgraders that rewrite the SDKv2 zero-value
// shape ("", [], {}) into framework null semantics.
//
// Background: the pre-v3 SDKv2 provider serialised unset Optional
// attributes as their Go zero value rather than as null. The framework
// port reads the same API responses but returns null for the same
// "user-didn't-set-this" case, producing spurious "[] -> null" /
// "\"\" -> null" plan diffs on first refresh against pre-v3 state. The
// upgraders here migrate state in-place once on first read so subsequent
// plans see the correct null shape and no diff.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// nullIfEmptyString returns a null types.String when v carries the empty
// string. Pass-through for null / unknown.
func nullIfEmptyString(v types.String) types.String {
	if v.IsNull() || v.IsUnknown() {
		return v
	}
	if v.ValueString() == "" {
		return types.StringNull()
	}
	return v
}

// nullIfEmptyList returns a typed null List when l is a non-null
// zero-length list. Element type is preserved from the input so the
// upgraded state matches the current schema's expected list shape.
func nullIfEmptyList(ctx context.Context, l types.List) types.List {
	if l.IsNull() || l.IsUnknown() {
		return l
	}
	if len(l.Elements()) == 0 {
		return types.ListNull(l.ElementType(ctx))
	}
	return l
}

// nullIfEmptySet is nullIfEmptyList for types.Set.
func nullIfEmptySet(ctx context.Context, s types.Set) types.Set {
	if s.IsNull() || s.IsUnknown() {
		return s
	}
	if len(s.Elements()) == 0 {
		return types.SetNull(s.ElementType(ctx))
	}
	return s
}

// csaObjectFromV0List projects a v0 (SDKv2) single-element
// client_side_availability / default_client_side_availability list into
// the v3 single-object shape. attrTypes selects the resource's inner
// attribute set (feature_flag vs project — both {using_environment_id,
// using_mobile_key}). Returns a null object for null/empty input.
func csaObjectFromV0List(ctx context.Context, l types.List, attrTypes map[string]attr.Type) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	if l.IsNull() || l.IsUnknown() || len(l.Elements()) == 0 {
		return types.ObjectNull(attrTypes), diags
	}
	type csaModel struct {
		UsingEnvironmentID types.Bool `tfsdk:"using_environment_id"`
		UsingMobileKey     types.Bool `tfsdk:"using_mobile_key"`
	}
	var models []csaModel
	diags.Append(l.ElementsAs(ctx, &models, false)...)
	if diags.HasError() || len(models) == 0 {
		return types.ObjectNull(attrTypes), diags
	}
	obj, d := types.ObjectValue(attrTypes, map[string]attr.Value{
		USING_ENVIRONMENT_ID: models[0].UsingEnvironmentID,
		USING_MOBILE_KEY:     models[0].UsingMobileKey,
	})
	diags.Append(d...)
	return obj, diags
}

// defaultsObjectFromV0List projects a v0 (SDKv2) single-element
// feature_flag defaults list into the v3 single-object shape. Returns a
// null object for null/empty input.
func defaultsObjectFromV0List(ctx context.Context, l types.List) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	if l.IsNull() || l.IsUnknown() || len(l.Elements()) == 0 {
		return types.ObjectNull(featureFlagDefaultsAttrTypes), diags
	}
	type defaultsModel struct {
		OnVariation  types.Int64 `tfsdk:"on_variation"`
		OffVariation types.Int64 `tfsdk:"off_variation"`
	}
	var models []defaultsModel
	diags.Append(l.ElementsAs(ctx, &models, false)...)
	if diags.HasError() || len(models) == 0 {
		return types.ObjectNull(featureFlagDefaultsAttrTypes), diags
	}
	obj, d := types.ObjectValue(featureFlagDefaultsAttrTypes, map[string]attr.Value{
		ON_VARIATION:  models[0].OnVariation,
		OFF_VARIATION: models[0].OffVariation,
	})
	diags.Append(d...)
	return obj, diags
}
