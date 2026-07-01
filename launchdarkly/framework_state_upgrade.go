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

// environmentsMapFromV0List re-keys a v0 (SDKv2 / pre-REL-14236)
// environments list — whose elements carried the env key inline — into the
// v3 map keyed by env key. Each per-env approval_settings that matches the
// API defaults the v2.29 SDKv2 provider persisted verbatim is collapsed to
// null so the v3 plan doesn't churn. Returns a null map for null/empty
// input.
func environmentsMapFromV0List(ctx context.Context, l types.List) (types.Map, diag.Diagnostics) {
	var diags diag.Diagnostics
	if l.IsNull() || l.IsUnknown() || len(l.Elements()) == 0 {
		// No environments in the v0 state: manage none. Return an EMPTY map,
		// not null — a null prior makes the next Read import every environment
		// (the null branch of environmentsMapFromAPI), which would undo
		// manage-none and risk later unintended deletes. (v2 required at least
		// one environment, so this branch is defensive.)
		m, d := types.MapValue(environmentObjectType, map[string]attr.Value{})
		diags.Append(d...)
		return m, diags
	}
	var v0envs []environmentModelV0
	diags.Append(l.ElementsAs(ctx, &v0envs, false)...)
	if diags.HasError() {
		return types.MapNull(environmentObjectType), diags
	}
	approvalElemType := types.ObjectType{AttrTypes: frameworkApprovalSettingsObjectAttrTypes}
	elements := make(map[string]attr.Value, len(v0envs))
	for _, e := range v0envs {
		approvals := e.ApprovalSettings
		if !approvals.IsNull() && !approvals.IsUnknown() && len(approvals.Elements()) == 1 {
			var items []approvalSettingsModel
			d := approvals.ElementsAs(ctx, &items, false)
			if !d.HasError() && len(items) == 1 && approvalSettingsMatchesAPIDefaults(items[0]) {
				approvals = types.ListNull(approvalElemType)
			}
		}
		obj, d := types.ObjectValue(environmentAttrTypes, map[string]attr.Value{
			KEY:                  e.Key,
			NAME:                 e.Name,
			COLOR:                e.Color,
			CRITICAL:             e.Critical,
			API_KEY:              e.APIKey,
			MOBILE_KEY:           e.MobileKey,
			CLIENT_SIDE_ID:       e.ClientSideID,
			DEFAULT_TTL:          e.DefaultTTL,
			SECURE_MODE:          e.SecureMode,
			DEFAULT_TRACK_EVENTS: e.DefaultTrackEvents,
			REQUIRE_COMMENTS:     e.RequireComments,
			CONFIRM_CHANGES:      e.ConfirmChanges,
			TAGS:                 e.Tags,
			APPROVAL_SETTINGS:    approvals,
		})
		diags.Append(d...)
		elements[e.Key.ValueString()] = obj
	}
	m, d := types.MapValue(environmentObjectType, elements)
	diags.Append(d...)
	return m, diags
}

// defaultsObjectFromV0List projects a v0 (SDKv2) single-element
// feature_flag defaults list into the v3 single-object shape. Returns a
// null object for null/empty input. The v0 schema predates the
// *_name/*_value alternatives to on_variation/off_variation (REL-14238),
// so those always come across null.
func defaultsObjectFromV0List(ctx context.Context, l types.List) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	if l.IsNull() || l.IsUnknown() || len(l.Elements()) == 0 {
		return types.ObjectNull(featureFlagResourceDefaultsAttrTypes), diags
	}
	type defaultsModel struct {
		OnVariation  types.Int64 `tfsdk:"on_variation"`
		OffVariation types.Int64 `tfsdk:"off_variation"`
	}
	var models []defaultsModel
	diags.Append(l.ElementsAs(ctx, &models, false)...)
	if diags.HasError() || len(models) == 0 {
		return types.ObjectNull(featureFlagResourceDefaultsAttrTypes), diags
	}
	obj, d := types.ObjectValue(featureFlagResourceDefaultsAttrTypes, map[string]attr.Value{
		ON_VARIATION:        models[0].OnVariation,
		ON_VARIATION_NAME:   types.StringNull(),
		ON_VARIATION_VALUE:  types.StringNull(),
		OFF_VARIATION:       models[0].OffVariation,
		OFF_VARIATION_NAME:  types.StringNull(),
		OFF_VARIATION_VALUE: types.StringNull(),
	})
	diags.Append(d...)
	return obj, diags
}
