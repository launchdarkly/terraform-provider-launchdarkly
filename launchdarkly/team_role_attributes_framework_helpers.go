package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func roleAttributesFromFrameworkMap(ctx context.Context, m types.Map) (map[string][]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if m.IsNull() || m.IsUnknown() {
		return nil, diags
	}
	raw := make(map[string][]string, len(m.Elements()))
	for k, v := range m.Elements() {
		list, ok := v.(types.List)
		if !ok {
			diags.AddError("Unexpected role_attributes element type",
				"Expected list of strings for role_attributes["+k+"].")
			continue
		}
		values := make([]string, 0, len(list.Elements()))
		d := list.ElementsAs(ctx, &values, false)
		diags.Append(d...)
		raw[k] = values
	}
	return raw, diags
}

func roleAttributesToFrameworkMap(raw *map[string][]string) (types.Map, diag.Diagnostics) {
	elemType := types.ListType{ElemType: types.StringType}
	if raw == nil || len(*raw) == 0 {
		return types.MapNull(elemType), nil
	}
	var diags diag.Diagnostics
	out := make(map[string]attr.Value, len(*raw))
	for k, values := range *raw {
		elems := make([]attr.Value, 0, len(values))
		for _, v := range values {
			elems = append(elems, types.StringValue(v))
		}
		listVal, d := types.ListValue(types.StringType, elems)
		diags.Append(d...)
		out[k] = listVal
	}
	mapVal, d := types.MapValue(elemType, out)
	diags.Append(d...)
	return mapVal, diags
}

// diffRoleAttributePatches returns a single replaceRoleAttributes instruction
// when existing and desired differ, or nil otherwise. We use the wholesale
// replacement form because it is atomic and idempotent against value-order
// differences from the API.
func diffRoleAttributePatches(existing, desired map[string][]string) []map[string]interface{} {
	if roleAttributesEqual(existing, desired) {
		return nil
	}
	value := desired
	if value == nil {
		value = map[string][]string{}
	}
	return []map[string]interface{}{
		{
			"kind":  "replaceRoleAttributes",
			"value": value,
		},
	}
}

func roleAttributesEqual(a, b map[string][]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok || !stringSlicesEqualUnordered(va, vb) {
			return false
		}
	}
	return true
}

func stringSlicesEqualUnordered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[string]int, len(a))
	for _, v := range a {
		counts[v]++
	}
	for _, v := range b {
		counts[v]--
		if counts[v] < 0 {
			return false
		}
	}
	return true
}
