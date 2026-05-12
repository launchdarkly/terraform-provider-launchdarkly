package launchdarkly

// policy_statements_framework.go is the terraform-plugin-framework analogue
// of policy_statements_helper.go. It provides:
//
//   - frameworkPolicyStatementsBlock: a ListNestedBlock schema producer
//     (data-source variant: all Computed). The block name (e.g.
//     POLICY_STATEMENTS, POLICY, STATEMENTS) is supplied by the caller so
//     the same builder works for relay_proxy_configuration (uses POLICY),
//     webhook (uses STATEMENTS), audit_log_subscription, etc.
//   - frameworkPolicyStatementsValue: converts []ldapi.Statement to a
//     framework types.List of nested objects, suitable for assignment to
//     the model field.
//
// SDKv2 source: policy_statements_helper.go (schema and
// policyStatementsToResourceData).

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

// frameworkPolicyStatementsObjectAttrTypes is the attribute-type map
// every list-nested-block element conforms to. Used both by the schema
// builder and by the converter when constructing types.ObjectValue.
var frameworkPolicyStatementsObjectAttrTypes = map[string]attr.Type{
	RESOURCES:     types.ListType{ElemType: types.StringType},
	NOT_RESOURCES: types.ListType{ElemType: types.StringType},
	ACTIONS:       types.ListType{ElemType: types.StringType},
	NOT_ACTIONS:   types.ListType{ElemType: types.StringType},
	EFFECT:        types.StringType,
}

// frameworkPolicyStatementsDataSourceBlock returns a ListNestedBlock
// schema for use in datasource.Schema. All inner attrs are Computed
// because data sources are read-only; the SDKv2 version distinguishes
// computed-vs-optional via the options struct.
func frameworkPolicyStatementsDataSourceBlock(description string) dsschema.ListNestedBlock {
	return dsschema.ListNestedBlock{
		Description: description,
		NestedObject: dsschema.NestedBlockObject{
			Attributes: map[string]dsschema.Attribute{
				RESOURCES: dsschema.ListAttribute{
					Computed:    true,
					ElementType: types.StringType,
					Description: "The list of resource specifiers defining the resources to which the statement applies.",
				},
				NOT_RESOURCES: dsschema.ListAttribute{
					Computed:    true,
					ElementType: types.StringType,
					Description: "The list of resource specifiers defining the resources to which the statement does not apply.",
				},
				ACTIONS: dsschema.ListAttribute{
					Computed:    true,
					ElementType: types.StringType,
					Description: "The list of action specifiers defining the actions to which the statement applies.",
				},
				NOT_ACTIONS: dsschema.ListAttribute{
					Computed:    true,
					ElementType: types.StringType,
					Description: "The list of action specifiers defining the actions to which the statement does not apply.",
				},
				EFFECT: dsschema.StringAttribute{
					Computed:    true,
					Description: "Either `allow` or `deny`. This argument defines whether the statement allows or denies access to the named resources and actions.",
				},
			},
		},
	}
}

// frameworkPolicyStatementsValue converts an LD-API []Statement into a
// framework types.List of objects matching
// frameworkPolicyStatementsObjectAttrTypes. Empty slice maps to an
// empty list (not null) so state writes stay deterministic across
// plans.
func frameworkPolicyStatementsValue(ctx context.Context, statements []ldapi.Statement) (basetypes.ListValue, diag.Diagnostics) {
	objectType := types.ObjectType{AttrTypes: frameworkPolicyStatementsObjectAttrTypes}

	elements := make([]attr.Value, 0, len(statements))
	var diags diag.Diagnostics

	for _, s := range statements {
		resources, d := listFromStringSlice(ctx, s.Resources)
		diags.Append(d...)
		notResources, d := listFromStringSlice(ctx, s.NotResources)
		diags.Append(d...)
		actions, d := listFromStringSlice(ctx, s.Actions)
		diags.Append(d...)
		notActions, d := listFromStringSlice(ctx, s.NotActions)
		diags.Append(d...)

		obj, d := types.ObjectValue(frameworkPolicyStatementsObjectAttrTypes, map[string]attr.Value{
			RESOURCES:     resources,
			NOT_RESOURCES: notResources,
			ACTIONS:       actions,
			NOT_ACTIONS:   notActions,
			EFFECT:        types.StringValue(s.Effect),
		})
		diags.Append(d...)
		elements = append(elements, obj)
	}

	list, d := types.ListValue(objectType, elements)
	diags.Append(d...)
	return list, diags
}
