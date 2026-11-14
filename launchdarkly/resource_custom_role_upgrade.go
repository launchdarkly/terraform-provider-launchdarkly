package launchdarkly

// Frozen pre-v3 custom_role schema + model used as PriorSchema for
// the v0->v1 state upgrader. The v0 shape (v2.x SDKv2 provider)
// carried the deprecated `policy` set; v3 drops it. The upgrader
// decodes prior state into CustomRoleResourceModelV0 and projects to
// the current CustomRoleResourceModel, mapping each policy element
// onto a policy_statements element when policy_statements was empty.

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type CustomRoleResourceModelV0 struct {
	ID                   types.String `tfsdk:"id"`
	Key                  types.String `tfsdk:"key"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	BasePermissions      types.String `tfsdk:"base_permissions"`
	Policy               types.Set    `tfsdk:"policy"`
	PolicyStatements     types.List   `tfsdk:"policy_statements"`
	PolicyStatementsJSON types.String `tfsdk:"policy_statements_json"`
}

func customRoleSchemaAttributesV0() map[string]schema.Attribute {
	attrs := customRoleSchemaAttributes()
	attrs[POLICY] = schema.SetNestedAttribute{
		Optional:           true,
		DeprecationMessage: "'policy' is now deprecated. Please migrate to 'policy_statements' to maintain future compatability.",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				RESOURCES: schema.ListAttribute{
					Required:    true,
					ElementType: types.StringType,
				},
				ACTIONS: schema.ListAttribute{
					Required:    true,
					ElementType: types.StringType,
				},
				EFFECT: schema.StringAttribute{
					Required: true,
				},
			},
		},
	}
	return attrs
}
