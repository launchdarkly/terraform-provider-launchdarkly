package launchdarkly

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
				{Name: "nameValue", Description: "descValue", Value: ptr("a string value")},
				{Name: "nameValue2", Description: "descValue2", Value: ptr("another string value")},
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
				map[string]*schema.Schema{VARIATION_TYPE: variationTypeSchema(), VARIATIONS: variationsSchema()},
				tc.vars,
			)

			actualVariations, err := variationsFromResourceData(resourceData)
			require.NoError(t, err)
			for idx, expected := range tc.expected {
				assert.Equal(t, expected.Name, actualVariations[idx].Name)
				assert.Equal(t, expected.Description, actualVariations[idx].Description)
				assert.Equal(t, *expected.Value, *actualVariations[idx].Value)
			}
		})
	}
}
