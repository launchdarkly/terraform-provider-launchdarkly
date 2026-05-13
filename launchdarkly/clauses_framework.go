package launchdarkly

// clauses_framework.go is the terraform-plugin-framework analogue of
// clause_helper.go's schema + conversion helpers. Clauses are
// nested inside segment rules and feature_flag/feature_flag_environment
// rules; sharing the helper avoids per-resource drift.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

// frameworkClausesDataSourceBlock returns the schema for a list-nested
// block of clauses suitable for use in datasource.Schema.
func frameworkClausesDataSourceBlock() dsschema.ListNestedBlock {
	return dsschema.ListNestedBlock{
		Description: "Clauses applied as the rule's logical condition.",
		NestedObject: dsschema.NestedBlockObject{
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
			if err == nil {
				valueType = vt
			}
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
