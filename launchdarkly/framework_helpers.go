package launchdarkly

// framework_helpers.go centralises the small utilities that every
// terraform-plugin-framework resource and data source in this provider needs:
// extracting the configured *Client, converting between framework
// types.Set / types.List and Go []string, surfacing LD API errors as
// framework diagnostics. Keep the surface narrow — anything that needs
// non-trivial logic belongs next to the resource that consumes it.

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// diagnosticsSink is the narrow interface satisfied by every framework
// *Response type (CreateResponse, ReadResponse, etc.) that carries a
// Diagnostics field. We accept the interface rather than a concrete type so
// helpers work uniformly across resource and data source surfaces.
type diagnosticsSink interface {
	AddError(summary, detail string)
}

func configureResourceClient(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *Client {
	if req.ProviderData == nil {
		return nil
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *launchdarkly.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return nil
	}
	return client
}

func configureDataSourceClient(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) *Client {
	if req.ProviderData == nil {
		return nil
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *launchdarkly.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return nil
	}
	return client
}

func addLdapiError(diags diagnosticsSink, summary string, err error) {
	if err == nil {
		return
	}
	diags.AddError(summary, handleLdapiErr(err).Error())
}

// stringSliceFromSet returns an empty (not nil) slice for null / unknown
// inputs so callers can use len() uniformly.
func stringSliceFromSet(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	if set.IsNull() || set.IsUnknown() {
		return []string{}, nil
	}
	out := make([]string, 0, len(set.Elements()))
	diags := set.ElementsAs(ctx, &out, false)
	return out, diags
}

// setFromStringSlice produces a non-null empty Set for nil input so state
// writes don't flip between null and empty across plans.
func setFromStringSlice(ctx context.Context, vals []string) (types.Set, diag.Diagnostics) {
	if len(vals) == 0 {
		return types.SetValueMust(types.StringType, []attr.Value{}), nil
	}
	return types.SetValueFrom(ctx, types.StringType, vals)
}

func stringSliceFromList(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return []string{}, nil
	}
	out := make([]string, 0, len(list.Elements()))
	diags := list.ElementsAs(ctx, &out, false)
	return out, diags
}

func listFromStringSlice(ctx context.Context, vals []string) (types.List, diag.Diagnostics) {
	if vals == nil {
		vals = []string{}
	}
	return types.ListValueFrom(ctx, types.StringType, vals)
}

// listFromStringSlicePreservingPlan is the list analogue of
// setFromStringSlicePreservingPlan: API empty + existing null → null;
// API empty + existing populated → empty list; API populated → list.
func listFromStringSlicePreservingPlan(ctx context.Context, vals []string, existing types.List) (types.List, diag.Diagnostics) {
	if len(vals) > 0 {
		return types.ListValueFrom(ctx, types.StringType, vals)
	}
	if existing.IsNull() {
		return types.ListNull(types.StringType), nil
	}
	return types.ListValueMust(types.StringType, []attr.Value{}), nil
}

// stringValueFromPointer dereferences a *string into a non-null
// types.String, defaulting to empty when the pointer is nil. Use for
// Computed attributes where "" is the intended state for an absent API
// field — Computed absorbs the plan-vs-apply mismatch null and "" would
// otherwise produce.
func stringValueFromPointer(s *string) types.String {
	if s == nil {
		return types.StringValue("")
	}
	return types.StringValue(*s)
}

// stringValueOrNull returns null when v == "", else types.StringValue(v).
// Use when mapping API responses that distinguish "absent" from "empty
// string" into framework state for Optional-only attributes.
func stringValueOrNull(v string) types.String {
	if v == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}

// stringValueOrNullFromPointer is stringValueOrNull for *string inputs:
// null when the pointer is nil or points to "". Use for Optional
// (non-Computed) string attributes; writing "" would trip
// terraform-core's plan-apply consistency check on plan-null configs.
func stringValueOrNullFromPointer(p *string) types.String {
	if p == nil {
		return types.StringNull()
	}
	return stringValueOrNull(*p)
}

// setFromStringSliceOrNull returns a null Set on empty input. Use for
// Optional (non-Computed) Set attributes whose user omits-the-attribute
// case must round-trip null. See setFromStringSlicePreservingPlan for
// the variant that also accommodates explicit `attr = []` HCL.
func setFromStringSliceOrNull(ctx context.Context, vals []string) (types.Set, diag.Diagnostics) {
	if len(vals) == 0 {
		return types.SetNull(types.StringType), nil
	}
	return types.SetValueFrom(ctx, types.StringType, vals)
}

// setFromStringSlicePreservingPlan preserves the user's null-vs-empty
// intent when the API returns nothing for an Optional Set attribute,
// using `existing` (the plan value during Create, state value during
// Read-refresh) to disambiguate:
//
//   - API has values → return the populated Set.
//   - API empty AND existing is null → null (user omitted the attribute).
//   - API empty AND existing is non-null → empty Set (covers explicit
//     `attr = []` plans and drift from populated → empty).
//
// Writing the wrong null/empty form trips terraform-core's plan-apply
// consistency check.
func setFromStringSlicePreservingPlan(ctx context.Context, vals []string, existing types.Set) (types.Set, diag.Diagnostics) {
	if len(vals) > 0 {
		return types.SetValueFrom(ctx, types.StringType, vals)
	}
	// Unknown (Optional+Computed plan value when user omits the attr)
	// and Null (Optional-only plan value when user omits the attr) both
	// mean "user did not declare this", so emit null in state.
	if existing.IsNull() || existing.IsUnknown() {
		return types.SetNull(types.StringType), nil
	}
	return types.SetValueMust(types.StringType, []attr.Value{}), nil
}

// stringPointerFromAttr is the inverse: null / unknown framework values
// project to a nil *string, suitable for ldapi optional-field patches.
func stringPointerFromAttr(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

// mapStringFromAttr converts a framework types.Map (String elements) into
// a Go map. Null / unknown returns an empty (non-nil) map.
func mapStringFromAttr(ctx context.Context, m types.Map) (map[string]string, diag.Diagnostics) {
	out := make(map[string]string, len(m.Elements()))
	if m.IsNull() || m.IsUnknown() {
		return out, nil
	}
	d := m.ElementsAs(ctx, &out, false)
	return out, d
}
