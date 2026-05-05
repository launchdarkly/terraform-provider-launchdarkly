package launchdarkly

import (
	"github.com/hashicorp/go-cty/cty"
)

func ctyObjectGetAttr(config cty.Value, attr string) cty.Value {
	if config.IsNull() || !config.IsKnown() {
		return cty.NullVal(cty.DynamicPseudoType)
	}
	ty := config.Type()
	if !ty.IsObjectType() || !ty.HasAttribute(attr) {
		return cty.NullVal(cty.DynamicPseudoType)
	}
	return config.GetAttr(attr)
}

func ctyValueListElements(v cty.Value) []cty.Value {
	if v.IsNull() || !v.IsKnown() {
		return nil
	}
	return v.AsValueSlice()
}

func ctyBoolTrue(v cty.Value) bool {
	if v.IsNull() || !v.IsKnown() || v.Type() != cty.Bool {
		return false
	}
	return v.True()
}

// rawConfigHasAnyAttr reports whether the raw cty config object's *type* on a *schema.ResourceData
// (or *schema.ResourceDiff) declares at least one of the given top-level attributes. Used to
// detect when an embedded provider (Upjet) has stripped deprecated attributes from the runtime
// schema so callers can avoid emitting fallback patches that would overwrite server-side state.
//
// The check is intentionally on the value's type, not its null-ness: helper/schema returns a
// null value of the schema's implied object type when no live config is available, but the type
// still carries the schema's attributes.
func rawConfigHasAnyAttr(config cty.Value, attrs ...string) bool {
	ty := config.Type()
	if ty == cty.NilType || !ty.IsObjectType() {
		return false
	}
	for _, a := range attrs {
		if ty.HasAttribute(a) {
			return true
		}
	}
	return false
}
