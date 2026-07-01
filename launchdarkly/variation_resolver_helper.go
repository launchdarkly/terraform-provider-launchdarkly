package launchdarkly

// variation_resolver_helper.go resolves a *_variation_name / *_variation_value
// reference against a flag's variations down to the integer index the LD API
// requires (REL-14238). Variations have no server-side stable/unique key —
// unlike environment `key`, name and value carry no uniqueness guarantee —
// so resolution must fail loudly on ambiguity rather than guess.

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	ldapi "github.com/launchdarkly/api-client-go/v22"
)

// resolvableVariation is a minimal projection of a flag variation, used so
// the resolver works the same whether the source is a live API response
// ([]ldapi.Variation) or a same-resource Terraform config list.
type resolvableVariation struct {
	Index int32
	Name  *string
	Value interface{}
}

// variationSelector holds the raw index/name/value a user configured for a
// single variation-reference site. At most one may be set.
type variationSelector struct {
	Index *int32
	Name  *string
	Value *string // raw text as configured, parsed per variationType before comparison
}

// resolveVariationIndex resolves a selector against a flag's variations.
// attrLabel identifies the site in error messages (e.g. "off_variation",
// "fallthrough.variation", "rules[2].variation").
func resolveVariationIndex(sel variationSelector, variations []resolvableVariation, attrLabel string) (int32, error) {
	if err := validateVariationSelectorExclusivity(sel, attrLabel); err != nil {
		return 0, err
	}
	switch {
	case sel.Index != nil:
		return *sel.Index, nil
	case sel.Name != nil:
		return matchVariationByName(*sel.Name, variations, attrLabel)
	case sel.Value != nil:
		return matchVariationByValue(*sel.Value, variations, attrLabel)
	default:
		return 0, fmt.Errorf("%s: exactly one of the index, _name, or _value form must be set", attrLabel)
	}
}

// validateVariationSelectorExclusivity checks "at most one set" without
// needing the variations list — usable at plan time (ValidateConfig) even
// when the variations live on a sibling resource this resource can't see.
func validateVariationSelectorExclusivity(sel variationSelector, attrLabel string) error {
	set := 0
	if sel.Index != nil {
		set++
	}
	if sel.Name != nil {
		set++
	}
	if sel.Value != nil {
		set++
	}
	if set > 1 {
		return fmt.Errorf("%s: only one of the index, _name, or _value form may be set", attrLabel)
	}
	return nil
}

func matchVariationByName(name string, variations []resolvableVariation, attrLabel string) (int32, error) {
	var matches []int32
	for _, v := range variations {
		if v.Name != nil && *v.Name == name {
			matches = append(matches, v.Index)
		}
	}
	switch len(matches) {
	case 0:
		return 0, fmt.Errorf("%s: no variation found with name %q", attrLabel, name)
	case 1:
		return matches[0], nil
	default:
		return 0, fmt.Errorf("%s: %d variations share the name %q; use the index form to disambiguate", attrLabel, len(matches), name)
	}
}

func matchVariationByValue(rawValue string, variations []resolvableVariation, attrLabel string) (int32, error) {
	variationType, err := variationsToVariationType(resolvableVariationsToAPI(variations))
	if err != nil {
		return 0, fmt.Errorf("%s: could not determine variation type: %w", attrLabel, err)
	}
	target, err := parseVariationRawValue(rawValue, variationType)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", attrLabel, err)
	}
	var matches []int32
	for _, v := range variations {
		if reflect.DeepEqual(v.Value, target) {
			matches = append(matches, v.Index)
		}
	}
	switch len(matches) {
	case 0:
		return 0, fmt.Errorf("%s: no variation found with value %s", attrLabel, rawValue)
	case 1:
		return matches[0], nil
	default:
		return 0, fmt.Errorf("%s: %d variations share that value; use the index form to disambiguate", attrLabel, len(matches))
	}
}

// parseVariationRawValue parses a *_variation_value string the same way
// variationFromTypedValue parses variations[].value for the given
// variation_type, but returns a plain (unwrapped) value so it can be
// reflect.DeepEqual-compared directly against ldapi.Variation.Value, which
// the API already delivers unwrapped.
func parseVariationRawValue(raw, variationType string) (interface{}, error) {
	switch variationType {
	case BOOL_VARIATION:
		return raw == "true", nil
	case STRING_VARIATION:
		return raw, nil
	case NUMBER_VARIATION:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("%q is an invalid number variation value: %w", raw, err)
		}
		return f, nil
	case JSON_VARIATION:
		var v interface{}
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return nil, fmt.Errorf("%q is an invalid json variation value: %w", raw, err)
		}
		return v, nil
	default:
		return nil, fmt.Errorf("invalid variation type: %q", variationType)
	}
}

// resolvableVariationsFromAPI projects ldapi.Variation values into
// resolvableVariation. Values from a live API response arrive already
// unwrapped by JSON decoding (bool/string/float64/map/slice), but values
// built locally via variationFromTypedValue (e.g. flagVariationsForResolution
// on the launchdarkly_feature_flag side, which never calls the real API)
// arrive pointer-wrapped (*bool/*float64/*interface{}) — dereference so
// both sources compare equally against a parsed *_variation_value.
func resolvableVariationsFromAPI(variations []ldapi.Variation) []resolvableVariation {
	out := make([]resolvableVariation, 0, len(variations))
	for i, v := range variations {
		out = append(out, resolvableVariation{Index: int32(i), Name: v.Name, Value: derefVariationValue(v.Value)})
	}
	return out
}

// derefVariationValue unwraps the pointer types variationFromTypedValue
// produces (*bool, *float64, *interface{}) down to the plain value
// underneath. A no-op for values that aren't pointers, so it's safe to
// apply unconditionally regardless of the caller's source.
func derefVariationValue(v interface{}) interface{} {
	switch t := v.(type) {
	case *bool:
		if t == nil {
			return nil
		}
		return *t
	case *float64:
		if t == nil {
			return nil
		}
		return *t
	case *string:
		if t == nil {
			return nil
		}
		return *t
	case *interface{}:
		if t == nil {
			return nil
		}
		return derefVariationValue(*t)
	default:
		return v
	}
}

// resolvableVariationsToAPI reprojects resolvableVariation back into
// []ldapi.Variation so existing helpers (variationsToVariationType) can be
// reused regardless of the resolver's input source.
func resolvableVariationsToAPI(variations []resolvableVariation) []ldapi.Variation {
	out := make([]ldapi.Variation, 0, len(variations))
	for _, v := range variations {
		out = append(out, ldapi.Variation{Name: v.Name, Value: v.Value})
	}
	return out
}
