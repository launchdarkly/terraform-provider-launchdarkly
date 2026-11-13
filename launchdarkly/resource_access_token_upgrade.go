package launchdarkly

// Frozen pre-v3 access_token schema + model used as PriorSchema for
// the v0->v1 state upgrader. The v0 shape (v2.x SDKv2 provider)
// carried the deprecated `expire` and `policy_statements`
// attributes; v3 drops both. The upgrader decodes prior state into
// AccessTokenResourceModelV0 and projects to the current
// AccessTokenResourceModel: expire is discarded (LD's public API
// never supported it), and policy_statements migrates verbatim into
// inline_roles (identical shape — both built from
// frameworkPolicyStatementsResourceAttribute).

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AccessTokenResourceModelV0 struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Role              types.String `tfsdk:"role"`
	CustomRoles       types.Set    `tfsdk:"custom_roles"`
	PolicyStatements  types.List   `tfsdk:"policy_statements"`
	InlineRoles       types.List   `tfsdk:"inline_roles"`
	ServiceToken      types.Bool   `tfsdk:"service_token"`
	DefaultAPIVersion types.Int64  `tfsdk:"default_api_version"`
	Token             types.String `tfsdk:"token"`
	Expire            types.Int64  `tfsdk:"expire"`
}

func accessTokenSchemaAttributesV0() map[string]schema.Attribute {
	attrs := accessTokenSchemaAttributes()
	attrs[EXPIRE] = schema.Int64Attribute{
		Optional:           true,
		Description:        "An expiration time for the current token secret, expressed as a Unix epoch time. Replace the computed token secret with a new value. The expired secret will no longer be able to authorize usage of the LaunchDarkly API. This field argument is **deprecated**. Please update your config to remove `expire` to maintain compatibility with future versions",
		DeprecationMessage: "'expire' is deprecated and will be removed in the next major release of the LaunchDarkly provider",
		Validators: []validator.Int64{
			noZeroValuesInt64Validator{},
		},
	}
	attrs[POLICY_STATEMENTS] = frameworkPolicyStatementsResourceAttribute(
		false,
		"Define inline custom roles. An array of statements with three attributes: effect, resources, actions. May be used in place of a built-in or custom role. This field argument is **deprecated**. Update your config to use `inline_role` to maintain compatibility with future versions.",
		"'policy_statements' is deprecated in favor of 'inline_roles'. This field will be removed in the next major release of the LaunchDarkly provider",
	)
	return attrs
}
