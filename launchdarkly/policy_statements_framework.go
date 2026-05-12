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
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	rsschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
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

// frameworkPolicyStatementsResourceBlock returns a ListNestedBlock for
// use in resource.Schema. The required flag controls whether the block
// itself is required; inner attrs default to Optional with element-list
// validation matching SDKv2.
func frameworkPolicyStatementsResourceBlock(required bool, description string, deprecated string) rsschema.ListNestedBlock {
	return rsschema.ListNestedBlock{
		Description:        description,
		DeprecationMessage: deprecated,
		NestedObject: rsschema.NestedBlockObject{
			Attributes: map[string]rsschema.Attribute{
				RESOURCES: rsschema.ListAttribute{
					Optional:    true,
					Computed:    true,
					ElementType: types.StringType,
					Description: "The list of resource specifiers the statement applies to.",
				},
				NOT_RESOURCES: rsschema.ListAttribute{
					Optional:    true,
					Computed:    true,
					ElementType: types.StringType,
					Description: "The list of resource specifiers the statement does not apply to.",
				},
				ACTIONS: rsschema.ListAttribute{
					Optional:    true,
					Computed:    true,
					ElementType: types.StringType,
					Description: "The list of actions the statement applies to.",
				},
				NOT_ACTIONS: rsschema.ListAttribute{
					Optional:    true,
					Computed:    true,
					ElementType: types.StringType,
					Description: "The list of actions the statement does not apply to.",
				},
				EFFECT: rsschema.StringAttribute{
					Required:    true,
					Description: "Either `allow` or `deny`.",
				},
			},
		},
	}
}

// frameworkPolicyStatementsFromList converts a framework types.List of
// policy-statement objects back into an []ldapi.StatementPost (used by
// resource Create/Update calls).
func frameworkPolicyStatementsFromList(ctx context.Context, list types.List) ([]ldapi.StatementPost, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}
	var elements []frameworkPolicyStatementModel
	diags.Append(list.ElementsAs(ctx, &elements, false)...)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]ldapi.StatementPost, 0, len(elements))
	for _, e := range elements {
		stmt, d := e.toLDAPI()
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		out = append(out, stmt)
	}
	return out, diags
}

type frameworkPolicyStatementModel struct {
	Resources    []string `tfsdk:"resources"`
	NotResources []string `tfsdk:"not_resources"`
	Actions      []string `tfsdk:"actions"`
	NotActions   []string `tfsdk:"not_actions"`
	Effect       string   `tfsdk:"effect"`
}

func (m frameworkPolicyStatementModel) toLDAPI() (ldapi.StatementPost, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(m.Resources) > 0 && len(m.NotResources) > 0 {
		diags.AddError("Invalid policy statement", errors.New("policy statements cannot contain both 'resources' and 'not_resources'").Error())
	}
	if len(m.Resources) == 0 && len(m.NotResources) == 0 {
		diags.AddError("Invalid policy statement", errors.New("policy statements must contain either 'resources' or 'not_resources'").Error())
	}
	if len(m.Actions) > 0 && len(m.NotActions) > 0 {
		diags.AddError("Invalid policy statement", errors.New("policy statements cannot contain both 'actions' and 'not_actions'").Error())
	}
	if len(m.Actions) == 0 && len(m.NotActions) == 0 {
		diags.AddError("Invalid policy statement", errors.New("policy statements must contain either 'actions' or 'not_actions'").Error())
	}
	if diags.HasError() {
		return ldapi.StatementPost{}, diags
	}
	stmt := ldapi.StatementPost{Effect: m.Effect}
	if len(m.Resources) > 0 {
		stmt.SetResources(m.Resources)
	}
	if len(m.NotResources) > 0 {
		stmt.SetNotResources(m.NotResources)
	}
	if len(m.Actions) > 0 {
		stmt.SetActions(m.Actions)
	}
	if len(m.NotActions) > 0 {
		stmt.SetNotActions(m.NotActions)
	}
	return stmt, diags
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
