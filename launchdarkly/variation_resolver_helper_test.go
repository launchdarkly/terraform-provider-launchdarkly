package launchdarkly

import (
	"reflect"
	"strings"
	"testing"

	ldapi "github.com/launchdarkly/api-client-go/v22"
)

func i32Ptr(i int32) *int32 { return &i }

func boolVariations() []resolvableVariation {
	return []resolvableVariation{
		{Index: 0, Name: strPtr("control"), Value: true},
		{Index: 1, Name: strPtr("treatment"), Value: false},
	}
}

func jsonVariations() []resolvableVariation {
	return []resolvableVariation{
		{Index: 0, Name: strPtr("a"), Value: map[string]interface{}{"x": float64(1), "y": float64(2)}},
		{Index: 1, Name: strPtr("b"), Value: map[string]interface{}{"z": float64(3)}},
	}
}

func TestResolveVariationIndex_ByIndex(t *testing.T) {
	idx, err := resolveVariationIndex(variationSelector{Index: i32Ptr(1)}, boolVariations(), "off_variation")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != 1 {
		t.Fatalf("expected index 1, got %d", idx)
	}
}

func TestResolveVariationIndex_ByName(t *testing.T) {
	idx, err := resolveVariationIndex(variationSelector{Name: strPtr("treatment")}, boolVariations(), "off_variation")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != 1 {
		t.Fatalf("expected index 1, got %d", idx)
	}
}

func TestResolveVariationIndex_ByValue_Bool(t *testing.T) {
	idx, err := resolveVariationIndex(variationSelector{Value: strPtr("false")}, boolVariations(), "off_variation")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != 1 {
		t.Fatalf("expected index 1, got %d", idx)
	}
}

func TestResolveVariationIndex_ByValue_JSONSemanticEquality(t *testing.T) {
	// Different key order/whitespace than stored, must still match via
	// decode+DeepEqual (same semantics as jsonNormalizePlanModifier).
	idx, err := resolveVariationIndex(variationSelector{Value: strPtr(`{ "y": 2,   "x": 1 }`)}, jsonVariations(), "fallthrough.variation")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != 0 {
		t.Fatalf("expected index 0, got %d", idx)
	}
}

func TestResolveVariationIndex_AmbiguousName(t *testing.T) {
	variations := []resolvableVariation{
		{Index: 0, Name: strPtr("dup"), Value: true},
		{Index: 1, Name: strPtr("dup"), Value: false},
	}
	_, err := resolveVariationIndex(variationSelector{Name: strPtr("dup")}, variations, "off_variation")
	if err == nil || !strings.Contains(err.Error(), "share the name") {
		t.Fatalf("expected ambiguous-name error, got %v", err)
	}
}

func TestResolveVariationIndex_AmbiguousValue(t *testing.T) {
	variations := []resolvableVariation{
		{Index: 0, Name: strPtr("a"), Value: true},
		{Index: 1, Name: strPtr("b"), Value: true},
	}
	_, err := resolveVariationIndex(variationSelector{Value: strPtr("true")}, variations, "off_variation")
	if err == nil || !strings.Contains(err.Error(), "share that value") {
		t.Fatalf("expected ambiguous-value error, got %v", err)
	}
}

func TestResolveVariationIndex_NoMatch(t *testing.T) {
	_, err := resolveVariationIndex(variationSelector{Name: strPtr("nonexistent")}, boolVariations(), "off_variation")
	if err == nil || !strings.Contains(err.Error(), "no variation found with name") {
		t.Fatalf("expected no-match error, got %v", err)
	}
}

func TestResolveVariationIndex_ConflictingIndexAndName(t *testing.T) {
	_, err := resolveVariationIndex(variationSelector{Index: i32Ptr(0), Name: strPtr("control")}, boolVariations(), "off_variation")
	if err == nil || !strings.Contains(err.Error(), "only one of") {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

func TestResolveVariationIndex_ConflictingNameAndValue(t *testing.T) {
	_, err := resolveVariationIndex(variationSelector{Name: strPtr("control"), Value: strPtr("true")}, boolVariations(), "off_variation")
	if err == nil || !strings.Contains(err.Error(), "only one of") {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

func TestResolveVariationIndex_NoneSet(t *testing.T) {
	_, err := resolveVariationIndex(variationSelector{}, boolVariations(), "off_variation")
	if err == nil || !strings.Contains(err.Error(), "must be set") {
		t.Fatalf("expected must-be-set error, got %v", err)
	}
}

func TestValidateVariationSelectorExclusivity_NoAPICallNeeded(t *testing.T) {
	if err := validateVariationSelectorExclusivity(variationSelector{Index: i32Ptr(0)}, "off_variation"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateVariationSelectorExclusivity(variationSelector{}, "off_variation"); err != nil {
		t.Fatalf("unexpected error for empty selector: %v", err)
	}
	err := validateVariationSelectorExclusivity(variationSelector{Index: i32Ptr(0), Value: strPtr("true")}, "off_variation")
	if err == nil || !strings.Contains(err.Error(), "only one of") {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

func TestParseVariationRawValue(t *testing.T) {
	cases := []struct {
		variationType string
		raw           string
		want          interface{}
	}{
		{BOOL_VARIATION, "true", true},
		{BOOL_VARIATION, "false", false},
		{STRING_VARIATION, "hello", "hello"},
		{NUMBER_VARIATION, "1.50", 1.5},
		{JSON_VARIATION, `{"a":1}`, map[string]interface{}{"a": float64(1)}},
	}
	for _, c := range cases {
		got, err := parseVariationRawValue(c.raw, c.variationType)
		if err != nil {
			t.Fatalf("%s/%s: unexpected error: %v", c.variationType, c.raw, err)
		}
		if m, ok := c.want.(map[string]interface{}); ok {
			gm, ok := got.(map[string]interface{})
			if !ok || len(gm) != len(m) {
				t.Fatalf("%s/%s: got %#v, want %#v", c.variationType, c.raw, got, c.want)
			}
			continue
		}
		if got != c.want {
			t.Fatalf("%s/%s: got %#v, want %#v", c.variationType, c.raw, got, c.want)
		}
	}
}

func TestParseVariationRawValue_InvalidNumber(t *testing.T) {
	if _, err := parseVariationRawValue("not-a-number", NUMBER_VARIATION); err == nil {
		t.Fatal("expected error for invalid number")
	}
}

func TestParseVariationRawValue_InvalidJSON(t *testing.T) {
	if _, err := parseVariationRawValue("{not json", JSON_VARIATION); err == nil {
		t.Fatal("expected error for invalid json")
	}
}

// TestResolvableVariationsFromAPI_DerefsPointerWrappedValues locks in a
// real bug found during acceptance-test authoring: variationFromTypedValue
// (used to build variations locally for launchdarkly_feature_flag.defaults,
// which never calls the real API) wraps values as *bool/*float64/*interface{}.
// A live GetFeatureFlag response never does this. resolvableVariationsFromAPI
// must normalize both shapes to the same plain value, or value-based
// resolution silently fails only on the feature_flag.defaults path.
func TestResolvableVariationsFromAPI_DerefsPointerWrappedValues(t *testing.T) {
	b := true
	f := 1.5
	var boxedString interface{} = "hello"
	var boxedJSON interface{} = map[string]interface{}{"a": float64(1)}
	variations := []ldapi.Variation{
		{Value: &b},
		{Value: &f},
		{Value: &boxedString},
		{Value: &boxedJSON},
		{Value: false}, // unwrapped, as a live API response would deliver it
	}
	resolvable := resolvableVariationsFromAPI(variations)
	wants := []interface{}{true, 1.5, "hello", map[string]interface{}{"a": float64(1)}, false}
	for i, want := range wants {
		if got := resolvable[i].Value; !reflect.DeepEqual(got, want) {
			t.Errorf("variation %d: got %#v (%T), want %#v", i, got, got, want)
		}
	}
}
