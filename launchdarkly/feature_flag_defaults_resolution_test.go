package launchdarkly

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

// These tests cover defaultsFromObject / flagVariationsForResolution
// (REL-14238) directly, without needing an API call — variations for
// launchdarkly_feature_flag.defaults live on the same resource, so
// resolution is a pure function of local config.

func namedBoolVariationsList(t *testing.T) types.List {
	t.Helper()
	objType := types.ObjectType{AttrTypes: featureFlagVariationAttrTypes}
	control, d := types.ObjectValue(featureFlagVariationAttrTypes, map[string]attr.Value{
		NAME:        types.StringValue("control"),
		DESCRIPTION: types.StringValue(""),
		VALUE:       types.StringValue("true"),
	})
	if d.HasError() {
		t.Fatalf("failed to build control variation: %v", d)
	}
	treatment, d := types.ObjectValue(featureFlagVariationAttrTypes, map[string]attr.Value{
		NAME:        types.StringValue("treatment"),
		DESCRIPTION: types.StringValue(""),
		VALUE:       types.StringValue("false"),
	})
	if d.HasError() {
		t.Fatalf("failed to build treatment variation: %v", d)
	}
	list, d := types.ListValue(objType, []attr.Value{control, treatment})
	if d.HasError() {
		t.Fatalf("failed to build variations list: %v", d)
	}
	return list
}

func defaultsObjectByIndex(t *testing.T, on, off int64) types.Object {
	t.Helper()
	obj, d := types.ObjectValue(featureFlagResourceDefaultsAttrTypes, map[string]attr.Value{
		ON_VARIATION:        types.Int64Value(on),
		ON_VARIATION_NAME:   types.StringNull(),
		ON_VARIATION_VALUE:  types.StringNull(),
		OFF_VARIATION:       types.Int64Value(off),
		OFF_VARIATION_NAME:  types.StringNull(),
		OFF_VARIATION_VALUE: types.StringNull(),
	})
	if d.HasError() {
		t.Fatalf("failed to build defaults object: %v", d)
	}
	return obj
}

func TestDefaultsFromObject_ByIndex(t *testing.T) {
	ctx := context.Background()
	apiVariations, diags := flagVariationsForResolution(ctx, namedBoolVariationsList(t), BOOL_VARIATION)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	defaults, diags := defaultsFromObject(ctx, defaultsObjectByIndex(t, 0, 1), apiVariations)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if defaults.OnVariation != 0 || defaults.OffVariation != 1 {
		t.Errorf("got on=%d off=%d, want on=0 off=1", defaults.OnVariation, defaults.OffVariation)
	}
}

func TestDefaultsFromObject_ByName(t *testing.T) {
	ctx := context.Background()
	apiVariations, diags := flagVariationsForResolution(ctx, namedBoolVariationsList(t), BOOL_VARIATION)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	obj, d := types.ObjectValue(featureFlagResourceDefaultsAttrTypes, map[string]attr.Value{
		ON_VARIATION:        types.Int64Null(),
		ON_VARIATION_NAME:   types.StringValue("control"),
		ON_VARIATION_VALUE:  types.StringNull(),
		OFF_VARIATION:       types.Int64Null(),
		OFF_VARIATION_NAME:  types.StringValue("treatment"),
		OFF_VARIATION_VALUE: types.StringNull(),
	})
	if d.HasError() {
		t.Fatalf("failed to build defaults object: %v", d)
	}
	defaults, diags := defaultsFromObject(ctx, obj, apiVariations)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if defaults.OnVariation != 0 || defaults.OffVariation != 1 {
		t.Errorf("got on=%d off=%d, want on=0 (control) off=1 (treatment)", defaults.OnVariation, defaults.OffVariation)
	}
}

func TestDefaultsFromObject_ByValue(t *testing.T) {
	ctx := context.Background()
	apiVariations, diags := flagVariationsForResolution(ctx, namedBoolVariationsList(t), BOOL_VARIATION)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	obj, d := types.ObjectValue(featureFlagResourceDefaultsAttrTypes, map[string]attr.Value{
		ON_VARIATION:        types.Int64Null(),
		ON_VARIATION_NAME:   types.StringNull(),
		ON_VARIATION_VALUE:  types.StringValue("true"),
		OFF_VARIATION:       types.Int64Null(),
		OFF_VARIATION_NAME:  types.StringNull(),
		OFF_VARIATION_VALUE: types.StringValue("false"),
	})
	if d.HasError() {
		t.Fatalf("failed to build defaults object: %v", d)
	}
	defaults, diags := defaultsFromObject(ctx, obj, apiVariations)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if defaults.OnVariation != 0 || defaults.OffVariation != 1 {
		t.Errorf("got on=%d off=%d, want on=0 (true) off=1 (false)", defaults.OnVariation, defaults.OffVariation)
	}
}

func TestDefaultsFromObject_NoMatchError(t *testing.T) {
	ctx := context.Background()
	apiVariations, diags := flagVariationsForResolution(ctx, namedBoolVariationsList(t), BOOL_VARIATION)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	obj, d := types.ObjectValue(featureFlagResourceDefaultsAttrTypes, map[string]attr.Value{
		ON_VARIATION:        types.Int64Value(0),
		ON_VARIATION_NAME:   types.StringNull(),
		ON_VARIATION_VALUE:  types.StringNull(),
		OFF_VARIATION:       types.Int64Null(),
		OFF_VARIATION_NAME:  types.StringValue("nonexistent"),
		OFF_VARIATION_VALUE: types.StringNull(),
	})
	if d.HasError() {
		t.Fatalf("failed to build defaults object: %v", d)
	}
	_, diags = defaultsFromObject(ctx, obj, apiVariations)
	if !diags.HasError() {
		t.Fatal("expected an error for a nonexistent off_variation_name")
	}
	found := false
	for _, dg := range diags {
		if strings.Contains(dg.Summary(), "off_variation") || strings.Contains(dg.Detail(), "no variation found") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected off_variation error, got %v", diags)
	}
}

func TestDefaultsFromObject_ConflictError(t *testing.T) {
	ctx := context.Background()
	apiVariations, diags := flagVariationsForResolution(ctx, namedBoolVariationsList(t), BOOL_VARIATION)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	obj, d := types.ObjectValue(featureFlagResourceDefaultsAttrTypes, map[string]attr.Value{
		ON_VARIATION:        types.Int64Value(0),
		ON_VARIATION_NAME:   types.StringValue("control"),
		ON_VARIATION_VALUE:  types.StringNull(),
		OFF_VARIATION:       types.Int64Value(1),
		OFF_VARIATION_NAME:  types.StringNull(),
		OFF_VARIATION_VALUE: types.StringNull(),
	})
	if d.HasError() {
		t.Fatalf("failed to build defaults object: %v", d)
	}
	_, diags = defaultsFromObject(ctx, obj, apiVariations)
	if !diags.HasError() {
		t.Fatal("expected a conflict error when both on_variation and on_variation_name are set")
	}
}

// TestFlagVariationsForResolution_BoolSynthesis locks in that resolution
// sees the same implicit [true, false] pair Create actually sends to the
// API for boolean flags with no explicit variations configured.
func TestFlagVariationsForResolution_BoolSynthesis(t *testing.T) {
	ctx := context.Background()
	variations, diags := flagVariationsForResolution(ctx, types.ListNull(types.ObjectType{AttrTypes: featureFlagVariationAttrTypes}), BOOL_VARIATION)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if len(variations) != 2 {
		t.Fatalf("expected synthesized [true, false], got %d variations", len(variations))
	}
	want := []ldapi.Variation{{Value: boolPtr(true)}, {Value: boolPtr(false)}}
	for i, v := range variations {
		if *(v.Value.(*bool)) != *(want[i].Value.(*bool)) {
			t.Errorf("variation %d: got %v, want %v", i, v.Value, want[i].Value)
		}
	}
}

func boolPtr(b bool) *bool { return &b }
