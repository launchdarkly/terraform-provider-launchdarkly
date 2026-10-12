package launchdarkly

// exclude_keys_from_state.go holds the shared pieces of the opt-in
// `exclude_keys_from_state` attribute on launchdarkly_environment and on the
// nested environments of launchdarkly_project. When the attribute is true the
// environment's secret keys (api_key and mobile_key) are stored as null in
// Terraform state instead of their plaintext values. client_side_id is not a
// secret (client-side SDKs ship it to browsers), so it is always stored.
//
// The attribute is Optional with no default: null (unset) and false behave
// identically and preserve the historical behavior, so existing states and
// configurations see no diff after upgrading.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const excludeKeysFromStateDescription = "Set to `true` to keep this environment's secret keys (`api_key` and `mobile_key`) out of Terraform state. When `true`, the provider stores `null` for these attributes instead of their values. `client_side_id` is not secret and is always stored. To use the keys elsewhere in your configuration without storing them, read them with the [`launchdarkly_environment_keys`](https://registry.terraform.io/providers/launchdarkly/launchdarkly/latest/docs/ephemeral-resources/environment_keys) ephemeral resource and pass them to write-only arguments. Changing this value updates the resource in place. It never replaces the environment or rotates its keys. Import cannot read your configuration, so an imported environment's keys are stored until the next apply with `exclude_keys_from_state = true` removes them. This field defaults to `false` when not set."

// excludeKeysFromStateEnabled reports whether the opt-in is explicitly
// enabled. Null and unknown values mean "not enabled".
func excludeKeysFromStateEnabled(v types.Bool) bool {
	return !v.IsNull() && !v.IsUnknown() && v.ValueBool()
}

// excludedKeyValue returns null when keys are excluded from state, and the
// API value otherwise.
func excludedKeyValue(exclude types.Bool, value string) types.String {
	if excludeKeysFromStateEnabled(exclude) {
		return types.StringNull()
	}
	return types.StringValue(value)
}

// nullWhenKeysExcludedFromState plans a null value for a secret key
// attribute when its sibling `exclude_keys_from_state` attribute is true.
// Without it the attribute would show as "(known after apply)" on every
// update even though apply always stores null.
func nullWhenKeysExcludedFromState() planmodifier.String {
	return nullWhenKeysExcludedModifier{}
}

type nullWhenKeysExcludedModifier struct{}

func (m nullWhenKeysExcludedModifier) Description(_ context.Context) string {
	return "Plans a null value when exclude_keys_from_state is true."
}

func (m nullWhenKeysExcludedModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m nullWhenKeysExcludedModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Resource is being destroyed.
	if req.Plan.Raw.IsNull() {
		return
	}
	var exclude types.Bool
	diags := req.Plan.GetAttribute(ctx, req.Path.ParentPath().AtName(EXCLUDE_KEYS_FROM_STATE), &exclude)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	if excludeKeysFromStateEnabled(exclude) {
		resp.PlanValue = types.StringNull()
	}
}
