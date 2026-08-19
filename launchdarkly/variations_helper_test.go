package launchdarkly

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	ldapi "github.com/launchdarkly/api-client-go/v22"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stringVariationsRaw builds the raw config shape for a string-typed variations block, one element per value.
func stringVariationsRaw(values []string) map[string]interface{} {
	vars := make([]interface{}, 0, len(values))
	for _, value := range values {
		vars = append(vars, map[string]interface{}{
			NAME:        "",
			DESCRIPTION: "",
			VALUE:       value,
		})
	}
	return map[string]interface{}{
		VARIATION_TYPE: "string",
		VARIATIONS:     vars,
	}
}

// TestVariationPatchesRemoveTrailingInDescendingOrder guards SWITCH-1412: when 2+ trailing variations are
// removed in a single apply, the remove ops must be emitted highest-index-first. JSON Patch applies ops
// sequentially against a shrinking array, so ascending removes (/variations/3 then /variations/4) reference
// an out-of-range index after the first op and fail with 400 invalid_patch.
func TestVariationPatchesRemoveTrailingInDescendingOrder(t *testing.T) {
	sm := map[string]*schema.Schema{
		VARIATION_TYPE: variationTypeSchema(),
		VARIATIONS:     variationsSchema(false),
	}
	r := &schema.Resource{Schema: sm}

	oldData := schema.TestResourceDataRaw(t, sm, stringVariationsRaw([]string{"a", "b", "c", "d", "e"}))
	oldData.SetId("example-flag")
	oldState := oldData.State()
	cfg := terraform.NewResourceConfigRaw(stringVariationsRaw([]string{"a", "b", "c"}))

	diff, err := r.Diff(context.Background(), oldState, cfg, nil)
	require.NoError(t, err)

	d, err := schema.InternalMap(sm).Data(oldState, diff)
	require.NoError(t, err)

	patches, err := variationPatchesFromResourceData(d)
	require.NoError(t, err)

	idx4 := -1
	idx3 := -1
	for i, p := range patches {
		switch p.Path {
		case "/variations/4":
			idx4 = i
		case "/variations/3":
			idx3 = i
		}
	}
	require.NotEqual(t, -1, idx4, "expected a remove op for /variations/4")
	require.NotEqual(t, -1, idx3, "expected a remove op for /variations/3")
	assert.Less(t, idx4, idx3, "remove /variations/4 must precede remove /variations/3")
}

func TestVariationsFromResourceData(t *testing.T) {
	testCases := []struct {
		name     string
		vars     map[string]interface{}
		expected []ldapi.Variation
	}{
		{
			name: "string variations",
			vars: map[string]interface{}{
				VARIATION_TYPE: "string",
				VARIATIONS: []interface{}{
					map[string]interface{}{
						NAME:        "nameValue",
						DESCRIPTION: "descValue",
						VALUE:       "a string value",
					},
					map[string]interface{}{
						NAME:        "nameValue2",
						DESCRIPTION: "descValue2",
						VALUE:       "another string value",
					},
				}},
			expected: []ldapi.Variation{
				{Name: strPtr("nameValue"), Description: strPtr("descValue"), Value: ptr("a string value")},
				{Name: strPtr("nameValue2"), Description: strPtr("descValue2"), Value: ptr("another string value")},
			},
		},
		{
			name: "boolean variations",
			vars: map[string]interface{}{
				VARIATION_TYPE: "boolean",
				VARIATIONS: []interface{}{
					map[string]interface{}{
						VALUE: "true",
					},
					map[string]interface{}{
						VALUE: "false",
					},
				}},
			expected: []ldapi.Variation{
				{Value: ptr(true)},
				{Value: ptr(false)},
			},
		},
		{
			name: "number variations",
			vars: map[string]interface{}{
				VARIATION_TYPE: "number",
				VARIATIONS: []interface{}{
					map[string]interface{}{
						VALUE: 32.5,
					},
					map[string]interface{}{
						VALUE: 12,
					},
					map[string]interface{}{
						VALUE: 0,
					},
				}},
			expected: []ldapi.Variation{
				{Value: ptr(float64(32.5))},
				{Value: ptr(float64(12))},
				{Value: ptr(float64(0))},
			},
		},
		{
			name: "json variations",
			vars: map[string]interface{}{
				VARIATION_TYPE: "json",
				VARIATIONS: []interface{}{
					map[string]interface{}{
						VALUE: `{ "foo": "bar" }`,
					},
					map[string]interface{}{
						VALUE: `{ "foo": "baz", "extra": {"nested": "json"} }`,
					},
				}},
			expected: []ldapi.Variation{
				{Value: ptr(map[string]interface{}{"foo": "bar"})},
				{Value: ptr(map[string]interface{}{
					"foo": "baz",
					"extra": map[string]interface{}{
						"nested": "json",
					},
				})},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resourceData := schema.TestResourceDataRaw(t,
				map[string]*schema.Schema{VARIATION_TYPE: variationTypeSchema(), VARIATIONS: variationsSchema(false)},
				tc.vars,
			)

			actualVariations, err := variationsFromResourceData(resourceData)
			require.NoError(t, err)
			for idx, expected := range tc.expected {
				assert.Equal(t, expected.Name, actualVariations[idx].Name)
				assert.Equal(t, expected.Description, actualVariations[idx].Description)
				assert.Equal(t, expected.Value, actualVariations[idx].Value)
			}
		})
	}
}
