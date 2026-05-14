package launchdarkly

// policy_statements_framework.go is the terraform-plugin-framework analogue
// of policy_statements_helper.go. It provides:
//
//   - frameworkPolicyStatementsDataSourceAttribute / frameworkPolicyStatementsResourceAttribute:
//     ListNestedAttribute schema producers (data-source variant: all
//     Computed). The attribute name (e.g. POLICY_STATEMENTS, POLICY,
//     STATEMENTS) is supplied by the caller so the same builder works for
//     relay_proxy_configuration, webhook, audit_log_subscription, etc.
//   - frameworkPolicyStatementsValue: converts []ldapi.Statement to a
//     framework types.List of nested objects, suitable for assignment to
//     the model field.
//
// SDKv2 source: policy_statements_helper.go (schema and
// policyStatementsToResourceData).

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	rsschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

// frameworkPolicyStatementsDataSourceAttribute returns a ListNestedAttribute
// schema for use in datasource.Schema. All inner attrs are Computed
// because data sources are read-only; the SDKv2 version distinguishes
// computed-vs-optional via the options struct.
func frameworkPolicyStatementsDataSourceAttribute(description string) dsschema.ListNestedAttribute {
	return dsschema.ListNestedAttribute{
		Computed:    true,
		Description: description,
		NestedObject: dsschema.NestedAttributeObject{
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

// frameworkPolicyStatementsResourceAttribute returns a ListNestedAttribute
// for use in resource.Schema. The required flag controls whether the
// attribute itself is required; inner attrs preserve the SDKv2 flag
// matrix (Optional + MinItems=1 via list-size validator). Inner
// descriptions and the effect enum validator mirror
// policy_statements_helper.go.
func frameworkPolicyStatementsResourceAttribute(required bool, description string, deprecated string) rsschema.ListNestedAttribute {
	attr := rsschema.ListNestedAttribute{
		Description:        description,
		DeprecationMessage: deprecated,
		NestedObject: rsschema.NestedAttributeObject{
			Attributes: map[string]rsschema.Attribute{
				RESOURCES: rsschema.ListAttribute{
					Optional:    true,
					ElementType: types.StringType,
					Description: "The list of resource specifiers defining the resources to which the statement applies.",
					Validators: []validator.List{
						listvalidator.SizeAtLeast(1),
					},
				},
				NOT_RESOURCES: rsschema.ListAttribute{
					Optional:    true,
					ElementType: types.StringType,
					Description: "The list of resource specifiers defining the resources to which the statement does not apply.",
					Validators: []validator.List{
						listvalidator.SizeAtLeast(1),
					},
				},
				ACTIONS: rsschema.ListAttribute{
					Optional:    true,
					ElementType: types.StringType,
					Description: "The list of action specifiers defining the actions to which the statement applies.\nEither `actions` or `not_actions` must be specified. For a list of available actions read [Actions reference](https://docs.launchdarkly.com/home/account-security/custom-roles/actions#actions-reference).",
					Validators: []validator.List{
						listvalidator.SizeAtLeast(1),
					},
				},
				NOT_ACTIONS: rsschema.ListAttribute{
					Optional:    true,
					ElementType: types.StringType,
					Description: "The list of action specifiers defining the actions to which the statement does not apply.",
					Validators: []validator.List{
						listvalidator.SizeAtLeast(1),
					},
				},
				EFFECT: rsschema.StringAttribute{
					Required:    true,
					Description: "Either `allow` or `deny`. This argument defines whether the statement allows or denies access to the named resources and actions.",
					Validators: []validator.String{
						oneOfValidator{allowed: []string{"allow", "deny"}},
					},
				},
			},
		},
	}
	if required {
		// SDKv2 emits MinItems=1 on the outer block when the schema is
		// not Computed (see policyStatementsSchema). Mirror that for
		// the resource variant via Required + list-size validator.
		attr.Required = true
		attr.Validators = []validator.List{listvalidator.SizeAtLeast(1)}
	} else {
		attr.Optional = true
	}
	return attr
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
// frameworkPolicyStatementsObjectAttrTypes. Empty inner slices project
// to null lists (matching SDKv2 Optional-only semantics where absent
// inner attrs are not present in state). When the API returns zero
// statements, return null so plan-vs-apply consistency holds for
// Optional-only callers (webhook, custom_role inline_roles, etc.).
func frameworkPolicyStatementsValue(ctx context.Context, statements []ldapi.Statement) (basetypes.ListValue, diag.Diagnostics) {
	objectType := types.ObjectType{AttrTypes: frameworkPolicyStatementsObjectAttrTypes}
	if len(statements) == 0 {
		return types.ListNull(objectType), nil
	}

	elements := make([]attr.Value, 0, len(statements))
	var diags diag.Diagnostics

	for _, s := range statements {
		resources := types.ListNull(types.StringType)
		if len(s.Resources) > 0 {
			v, d := listFromStringSlice(ctx, s.Resources)
			diags.Append(d...)
			resources = v
		}
		notResources := types.ListNull(types.StringType)
		if len(s.NotResources) > 0 {
			v, d := listFromStringSlice(ctx, s.NotResources)
			diags.Append(d...)
			notResources = v
		}
		actions := types.ListNull(types.StringType)
		if len(s.Actions) > 0 {
			v, d := listFromStringSlice(ctx, s.Actions)
			diags.Append(d...)
			actions = v
		}
		notActions := types.ListNull(types.StringType)
		if len(s.NotActions) > 0 {
			v, d := listFromStringSlice(ctx, s.NotActions)
			diags.Append(d...)
			notActions = v
		}

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
