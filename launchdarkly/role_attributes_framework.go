package launchdarkly

// role_attributes_framework.go is the terraform-plugin-framework analogue
// of role_attributes_helper.go's schema + conversion helpers. The
// role_attributes block is shared across launchdarkly_team data source +
// resource (Phase 3.7).

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var frameworkRoleAttributeAttrTypes = map[string]attr.Type{
	KEY:    types.StringType,
	VALUES: types.ListType{ElemType: types.StringType},
}

// frameworkRoleAttributesDataSourceBlock returns a SetNestedBlock
// schema mirroring the SDKv2 TypeSet of role_attribute objects.
func frameworkRoleAttributesDataSourceBlock() dsschema.SetNestedBlock {
	return dsschema.SetNestedBlock{
		Description: "Role attributes for the team. Keyed by attribute name with a list of resource-key values.",
		NestedObject: dsschema.NestedBlockObject{
			Attributes: map[string]dsschema.Attribute{
				KEY: dsschema.StringAttribute{
					Computed:    true,
					Description: "The role attribute key.",
				},
				VALUES: dsschema.ListAttribute{
					Computed:    true,
					ElementType: types.StringType,
					Description: "List of resource-key values for the attribute.",
				},
			},
		},
	}
}

// frameworkRoleAttributesValue converts an LD-API role_attributes map
// (map[key] -> []string values) into a framework types.Set of objects.
// Nil input returns an empty set.
func frameworkRoleAttributesValue(ctx context.Context, roleAttributes *map[string][]string) (basetypes.SetValue, diag.Diagnostics) {
	objectType := types.ObjectType{AttrTypes: frameworkRoleAttributeAttrTypes}
	if roleAttributes == nil {
		return types.SetValue(objectType, []attr.Value{})
	}

	var diags diag.Diagnostics
	elements := make([]attr.Value, 0, len(*roleAttributes))
	for key, values := range *roleAttributes {
		valuesList, d := listFromStringSlice(ctx, values)
		diags.Append(d...)
		obj, d := types.ObjectValue(frameworkRoleAttributeAttrTypes, map[string]attr.Value{
			KEY:    types.StringValue(key),
			VALUES: valuesList,
		})
		diags.Append(d...)
		elements = append(elements, obj)
	}
	set, d := types.SetValue(objectType, elements)
	diags.Append(d...)
	return set, diags
}
