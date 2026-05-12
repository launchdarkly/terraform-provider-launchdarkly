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
	rsschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	ldapi "github.com/launchdarkly/api-client-go/v22"
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

// frameworkRoleAttributesResourceBlock returns a SetNestedBlock for
// use in resource.Schema.
func frameworkRoleAttributesResourceBlock() rsschema.SetNestedBlock {
	return rsschema.SetNestedBlock{
		Description: "Role attributes for the resource. Keyed by attribute name with a list of resource-key values.",
		NestedObject: rsschema.NestedBlockObject{
			Attributes: map[string]rsschema.Attribute{
				KEY: rsschema.StringAttribute{
					Required:    true,
					Description: "The role attribute key.",
				},
				VALUES: rsschema.ListAttribute{
					Required:    true,
					ElementType: types.StringType,
					Description: "List of resource-key values for the attribute.",
				},
			},
		},
	}
}

type frameworkRoleAttributeModel struct {
	Key    string   `tfsdk:"key"`
	Values []string `tfsdk:"values"`
}

// frameworkRoleAttributesFromSet converts a framework types.Set of
// role_attribute objects back into the LD-API map[string][]string
// shape used by NewMemberForm.RoleAttributes etc.
func frameworkRoleAttributesFromSet(ctx context.Context, set types.Set) (*map[string][]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if set.IsNull() || set.IsUnknown() || len(set.Elements()) == 0 {
		return nil, diags
	}
	var entries []frameworkRoleAttributeModel
	diags.Append(set.ElementsAs(ctx, &entries, false)...)
	if diags.HasError() {
		return nil, diags
	}
	out := make(map[string][]string, len(entries))
	for _, e := range entries {
		out[e.Key] = append(out[e.Key], e.Values...)
	}
	return &out, diags
}

// frameworkRoleAttributePatches generates the patch operations to
// replace /roleAttributes on the server. Matches getRoleAttributePatches
// from role_attributes_helper.go.
func frameworkRoleAttributePatches(ctx context.Context, planSet, stateSet types.Set) []ldapi.PatchOperation {
	if planSet.Equal(stateSet) {
		return nil
	}
	plan, _ := frameworkRoleAttributesFromSet(ctx, planSet)
	if plan != nil {
		return []ldapi.PatchOperation{patchReplace("/roleAttributes", plan)}
	}
	return []ldapi.PatchOperation{patchReplace("/roleAttributes", make(map[string][]string))}
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
