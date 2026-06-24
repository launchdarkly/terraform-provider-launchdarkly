package launchdarkly

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// boolVarList builds a two-element variations list (true/false) with the given names and empty
// descriptions, matching the framework object shape used by the feature_flag resource.
func boolVarList(names [2]string) types.List {
	objType := types.ObjectType{AttrTypes: featureFlagVariationAttrTypes}
	mk := func(name, value string) attr.Value {
		return types.ObjectValueMust(featureFlagVariationAttrTypes, map[string]attr.Value{
			NAME:        types.StringValue(name),
			DESCRIPTION: types.StringValue(""),
			VALUE:       types.StringValue(value),
		})
	}
	return types.ListValueMust(objType, []attr.Value{mk(names[0], "true"), mk(names[1], "false")})
}

// TestVariationPatchesPreserveUnsetNameDescription guards the fix that keeps the migration lossless for
// boolean flags whose variation name/description were set outside Terraform: when the config omits them,
// the Update must not emit a name/description patch (which would clear the server value), but must still
// patch the required value. When the config sets a name, it must be patched.
func TestVariationPatchesPreserveUnsetNameDescription(t *testing.T) {
	ctx := context.Background()
	old := boolVarList([2]string{"Enabled", "Disabled"})

	patches, diags := variationPatchesFromLists(ctx, old, boolVarList([2]string{"", ""}), BOOL_VARIATION)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	sawValue := false
	for _, p := range patches {
		if strings.HasSuffix(p.Path, "/name") || strings.HasSuffix(p.Path, "/description") {
			t.Errorf("omitted name/description must not be patched, got path %s", p.Path)
		}
		if strings.HasSuffix(p.Path, "/value") {
			sawValue = true
		}
	}
	if !sawValue {
		t.Error("value is required and must still be patched")
	}

	patches, _ = variationPatchesFromLists(ctx, old, boolVarList([2]string{"On", "Off"}), BOOL_VARIATION)
	sawName := false
	for _, p := range patches {
		if strings.HasSuffix(p.Path, "/variations/0/name") {
			sawName = true
		}
	}
	if !sawName {
		t.Error("a configured variation name must be patched")
	}
}
