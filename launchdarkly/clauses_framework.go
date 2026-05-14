package launchdarkly

// clauses_framework.go provides the shared clauses schema + conversion
// helpers. Clauses are nested inside segment rules and feature_flag /
// feature_flag_environment rules; sharing the helper avoids per-resource
// drift.

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var frameworkClauseAttrTypes = map[string]attr.Type{
	ATTRIBUTE:    types.StringType,
	OP:           types.StringType,
	VALUES:       types.ListType{ElemType: types.StringType},
	VALUE_TYPE:   types.StringType,
	NEGATE:       types.BoolType,
	CONTEXT_KIND: types.StringType,
}

// frameworkClausesDataSourceAttribute returns the schema for a list-nested
// attribute of clauses suitable for use in datasource.Schema.
func frameworkClausesDataSourceAttribute() dsschema.ListNestedAttribute {
	return dsschema.ListNestedAttribute{
		Computed:    true,
		Description: "Clauses applied as the rule's logical condition.",
		NestedObject: dsschema.NestedAttributeObject{
			Attributes: map[string]dsschema.Attribute{
				ATTRIBUTE: dsschema.StringAttribute{Computed: true, Description: "User attribute to operate on."},
				OP:        dsschema.StringAttribute{Computed: true, Description: "The operator associated with the rule clause. Available options are `in`, `endsWith`, `startsWith`, `matches`, `contains`, `lessThan`, `greaterThan`, `lessThanOrEqual`, `greaterThanOrEqual`, `before`, `after`, `segmentMatch`, `semVerEqual`, `semVerLessThan`, and `semVerGreaterThan`. To learn more, read [Operators](https://docs.launchdarkly.com/sdk/concepts/flag-evaluation-rules#operators)."},
				VALUES: dsschema.ListAttribute{
					Computed:    true,
					ElementType: types.StringType,
					Description: "Values for the clause.",
				},
				VALUE_TYPE:   dsschema.StringAttribute{Computed: true, Description: "Type of each clause value (boolean / string / number)."},
				NEGATE:       dsschema.BoolAttribute{Computed: true, Description: "Whether to negate the clause."},
				CONTEXT_KIND: dsschema.StringAttribute{Computed: true, Description: "Context kind for the clause."},
			},
		},
	}
}

// frameworkClausesResourceAttribute returns the resource-side
// ListNestedAttribute for clauses.
func frameworkClausesResourceAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Required:    true,
		Description: "List of clauses specifying the logical conditions to evaluate",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				ATTRIBUTE: schema.StringAttribute{
					Required:    true,
					Description: "The user attribute to operate on",
				},
				OP: schema.StringAttribute{
					Required:    true,
					Description: "The operator associated with the rule clause. Available options are `in`, `endsWith`, `startsWith`, `matches`, `contains`, `lessThan`, `lessThanOrEqual`, `greaterThanOrEqual`, `before`, `after`, `segmentMatch`, `semVerEqual`, `semVerLessThan`, and `semVerGreaterThan`. Read LaunchDarkly's [Operators](https://docs.launchdarkly.com/sdk/concepts/flag-evaluation-rules#operators) documentation for more information.",
					Validators:  []validator.String{opValidator()},
				},
				VALUES: schema.ListAttribute{
					Required:    true,
					ElementType: types.StringType,
					Description: "The list of values associated with the rule clause.",
				},
				VALUE_TYPE: schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Default:     stringdefault.StaticString(STRING_CLAUSE_VALUE),
					Description: "The type for each of the clause's values. Available types are `boolean`, `string`, and `number`. If omitted, `value_type` defaults to `string`.",
					Validators: []validator.String{
						oneOfValidator{allowed: []string{BOOL_CLAUSE_VALUE, STRING_CLAUSE_VALUE, NUMBER_CLAUSE_VALUE}},
					},
				},
				NEGATE: schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "Whether to negate the rule clause.",
				},
				CONTEXT_KIND: schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Default:     stringdefault.StaticString("user"),
					Description: "The context kind associated with this rule clause. If omitted, defaults to `user`.",
				},
			},
		},
	}
}

// frameworkClausesFromList converts a framework ListValue of clauses
// into []ldapi.Clause. Mirrors clauseFromResourceData. Values are
// coerced to their typed forms based on value_type.
func frameworkClausesFromList(ctx context.Context, list types.List) ([]ldapi.Clause, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return []ldapi.Clause{}, diags
	}
	type clauseModel struct {
		Attribute   types.String `tfsdk:"attribute"`
		Op          types.String `tfsdk:"op"`
		Values      types.List   `tfsdk:"values"`
		ValueType   types.String `tfsdk:"value_type"`
		Negate      types.Bool   `tfsdk:"negate"`
		ContextKind types.String `tfsdk:"context_kind"`
	}
	var models []clauseModel
	d := list.ElementsAs(ctx, &models, false)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]ldapi.Clause, 0, len(models))
	for _, m := range models {
		valueType := m.ValueType.ValueString()
		if valueType == "" {
			valueType = STRING_CLAUSE_VALUE
		}
		rawValues, d := stringSliceFromList(ctx, m.Values)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		typedValues := make([]interface{}, 0, len(rawValues))
		for _, s := range rawValues {
			v, err := clauseValueFromString(s, valueType)
			if err != nil {
				diags.AddError(fmt.Sprintf("invalid clause value: %v", err), "")
				return nil, diags
			}
			typedValues = append(typedValues, v)
		}
		clause := ldapi.Clause{
			Attribute: m.Attribute.ValueString(),
			Op:        m.Op.ValueString(),
			Values:    typedValues,
			Negate:    m.Negate.ValueBool(),
		}
		ck := m.ContextKind.ValueString()
		if ck != "" {
			clause.ContextKind = &ck
		}
		out = append(out, clause)
	}
	return out, diags
}

// clauseValueFromString is the framework analogue of
// clauseValueFromResourceData, decoupled from *schema.ResourceData.
func clauseValueFromString(s, valueType string) (interface{}, error) {
	switch valueType {
	case STRING_CLAUSE_VALUE:
		return s, nil
	case BOOL_CLAUSE_VALUE:
		return convertBoolStringToBool(s)
	case NUMBER_CLAUSE_VALUE:
		return convertNumberStringToFloat(s)
	}
	return nil, fmt.Errorf("invalid clause value type %q", valueType)
}

// frameworkClausesValue converts an LD-API []Clause to a framework
// types.List of objects matching frameworkClauseAttrTypes.
func frameworkClausesValue(ctx context.Context, clauses []ldapi.Clause) (basetypes.ListValue, diag.Diagnostics) {
	objectType := types.ObjectType{AttrTypes: frameworkClauseAttrTypes}

	var diags diag.Diagnostics
	elements := make([]attr.Value, 0, len(clauses))
	for _, c := range clauses {
		var valueType string
		strValues := make([]string, 0, len(c.Values))
		for _, v := range c.Values {
			vt, err := inferClauseValueTypeFromValue(v)
			if err != nil {
				diags.AddWarning(
					"Unknown clause value type",
					fmt.Sprintf("clause value %v has an unsupported type; defaulting to %q", v, STRING_CLAUSE_VALUE),
				)
				vt = STRING_CLAUSE_VALUE
			}
			valueType = vt
			strValues = append(strValues, stringifyValue(v))
		}
		if valueType == "" {
			valueType = STRING_CLAUSE_VALUE
		}
		valuesList, d := listFromStringSlice(ctx, strValues)
		diags.Append(d...)
		contextKind := "user"
		if c.ContextKind != nil {
			contextKind = *c.ContextKind
		}
		obj, d := types.ObjectValue(frameworkClauseAttrTypes, map[string]attr.Value{
			ATTRIBUTE:    types.StringValue(c.Attribute),
			OP:           types.StringValue(c.Op),
			VALUES:       valuesList,
			VALUE_TYPE:   types.StringValue(valueType),
			NEGATE:       types.BoolValue(c.Negate),
			CONTEXT_KIND: types.StringValue(contextKind),
		})
		diags.Append(d...)
		elements = append(elements, obj)
	}
	list, d := types.ListValue(objectType, elements)
	diags.Append(d...)
	return list, diags
}
