package launchdarkly

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/stretchr/testify/require"
)

func TestCtyObjectGetAttr_missingAttributeReturnsNull(t *testing.T) {
	t.Parallel()

	config := cty.EmptyObjectVal
	v := ctyObjectGetAttr(config, INCLUDE_IN_SNIPPET)
	require.True(t, v.IsNull())

	v2 := ctyObjectGetAttr(config, DEFAULT_CLIENT_SIDE_AVAILABILITY)
	require.True(t, v2.IsNull())
}

func TestCtyObjectGetAttr_presentAttribute(t *testing.T) {
	t.Parallel()

	config := cty.ObjectVal(map[string]cty.Value{
		INCLUDE_IN_SNIPPET: cty.BoolVal(true),
	})
	v := ctyObjectGetAttr(config, INCLUDE_IN_SNIPPET)
	require.False(t, v.IsNull())
	require.True(t, v.True())
}

func TestCtyValueListElements_nullOrEmpty(t *testing.T) {
	t.Parallel()

	require.Nil(t, ctyValueListElements(cty.NullVal(cty.List(cty.String))))
	l := cty.ListVal([]cty.Value{cty.StringVal("a")})
	require.Len(t, ctyValueListElements(l), 1)
}

func TestCtyBoolTrue(t *testing.T) {
	t.Parallel()

	require.False(t, ctyBoolTrue(cty.NullVal(cty.Bool)))
	require.False(t, ctyBoolTrue(cty.StringVal("x")))
	require.True(t, ctyBoolTrue(cty.True))
	require.False(t, ctyBoolTrue(cty.False))
}

func TestRawConfigHasAnyAttr(t *testing.T) {
	t.Parallel()

	require.False(t, rawConfigHasAnyAttr(cty.NullVal(cty.DynamicPseudoType), "x"))
	require.False(t, rawConfigHasAnyAttr(cty.EmptyObjectVal, "x"))
	require.False(t, rawConfigHasAnyAttr(cty.StringVal("nope"), "x"))

	objA := cty.ObjectVal(map[string]cty.Value{"a": cty.BoolVal(true)})
	require.True(t, rawConfigHasAnyAttr(objA, "a"))
	require.False(t, rawConfigHasAnyAttr(objA, "b"))
	require.True(t, rawConfigHasAnyAttr(objA, "missing", "a"))

	// Null value of an object type still carries the type's attribute set.
	nullTyped := cty.NullVal(cty.Object(map[string]cty.Type{"a": cty.Bool}))
	require.True(t, rawConfigHasAnyAttr(nullTyped, "a"))
	require.False(t, rawConfigHasAnyAttr(nullTyped, "b"))
}
