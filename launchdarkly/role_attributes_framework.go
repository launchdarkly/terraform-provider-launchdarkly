package launchdarkly

// role_attributes_framework.go provides the shared role_attributes
// schema builder and conversion helpers. Used by the team / team_member
// resources and data sources.
//
// HCL surface: `role_attributes = { <key> = [<values>...] }` — a map of
// string lists keyed by the role attribute key, matching the LD-API
// shape and the launchdarkly_team_role_mapping resource. Modeled as a
// Set of {key, values} objects through 3.0.0-beta.4.

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

const roleAttributesDescription = "A map of role attributes, keyed by the role attribute key with a string array of resource keys as each value. For example, if your policy statement defines the resource `\"proj/$${roleAttribute/testAttribute}\"`, the key would be `testAttribute` and the values the keys of the projects you wanted to assign access to."

// frameworkRoleAttributesDataSourceAttribute returns a MapAttribute
// schema for role_attributes.
func frameworkRoleAttributesDataSourceAttribute() dsschema.MapAttribute {
	return dsschema.MapAttribute{
		Computed:    true,
		ElementType: types.ListType{ElemType: types.StringType},
		Description: roleAttributesDescription,
	}
}

// frameworkRoleAttributesResourceAttribute returns a MapAttribute for
// use in resource.Schema.
func frameworkRoleAttributesResourceAttribute() rsschema.MapAttribute {
	return rsschema.MapAttribute{
		Optional:    true,
		ElementType: types.ListType{ElemType: types.StringType},
		Description: roleAttributesDescription,
	}
}

// frameworkRoleAttributesFromMap converts the framework
// Map<String, List<String>> back into the LD-API map[string][]string
// shape used by NewMemberForm.RoleAttributes etc. Returns nil for a
// null/unknown/empty map.
func frameworkRoleAttributesFromMap(ctx context.Context, m types.Map) (*map[string][]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if m.IsNull() || m.IsUnknown() || len(m.Elements()) == 0 {
		return nil, diags
	}
	raw, d := roleAttributesFromFrameworkMap(ctx, m)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	return &raw, diags
}

// frameworkRoleAttributePatches generates the patch operations to
// replace /roleAttributes on the server. A null plan is a deliberate
// clear; an Unknown plan carries no user intent, so it is a no-op
// (defensive; a non-computed attribute is concrete by apply time).
func frameworkRoleAttributePatches(ctx context.Context, planMap, stateMap types.Map) []ldapi.PatchOperation {
	if planMap.IsUnknown() || planMap.Equal(stateMap) {
		return nil
	}
	plan, _ := frameworkRoleAttributesFromMap(ctx, planMap)
	if plan != nil {
		return []ldapi.PatchOperation{patchReplace("/roleAttributes", plan)}
	}
	return []ldapi.PatchOperation{patchReplace("/roleAttributes", make(map[string][]string))}
}

// frameworkRoleAttributesValue converts an LD-API role_attributes map
// (map[key] -> []string values) into a framework Map<String, List<String>>.
// Nil or empty input returns a null map so plan-vs-apply consistency
// holds for the resource variant (Optional-only schema).
func frameworkRoleAttributesValue(_ context.Context, roleAttributes *map[string][]string) (basetypes.MapValue, diag.Diagnostics) {
	return roleAttributesToFrameworkMap(roleAttributes)
}

// roleAttributesMapFromV0Set projects the v0 (pre-map) Set of
// {key, values} role_attribute objects into the current
// Map<String, List<String>> keyed by the role attribute key. Returns a
// null map for null/empty input.
func roleAttributesMapFromV0Set(ctx context.Context, set types.Set) (types.Map, diag.Diagnostics) {
	elemType := types.ListType{ElemType: types.StringType}
	var diags diag.Diagnostics
	if set.IsNull() || set.IsUnknown() || len(set.Elements()) == 0 {
		return types.MapNull(elemType), diags
	}
	type roleAttributeModelV0 struct {
		Key    types.String `tfsdk:"key"`
		Values types.List   `tfsdk:"values"`
	}
	var entries []roleAttributeModelV0
	diags.Append(set.ElementsAs(ctx, &entries, false)...)
	if diags.HasError() {
		return types.MapNull(elemType), diags
	}
	elements := make(map[string]attr.Value, len(entries))
	for _, e := range entries {
		elements[e.Key.ValueString()] = e.Values
	}
	m, d := types.MapValue(elemType, elements)
	diags.Append(d...)
	return m, diags
}

// roleAttributesSetAttributeV0 reproduces the pre-map role_attributes
// shape — a set of {key, values} objects — used only in PriorSchema
// declarations for v0/v1 state upgraders.
func roleAttributesSetAttributeV0() rsschema.SetNestedAttribute {
	return rsschema.SetNestedAttribute{
		Optional: true,
		NestedObject: rsschema.NestedAttributeObject{
			Attributes: map[string]rsschema.Attribute{
				KEY: rsschema.StringAttribute{Required: true},
				VALUES: rsschema.ListAttribute{
					Required:    true,
					ElementType: types.StringType,
				},
			},
		},
	}
}
