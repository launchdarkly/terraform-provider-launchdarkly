package launchdarkly

import (
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
	"github.com/stretchr/testify/assert"
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
				variation_type: "string",
				variations: []interface{}{
					map[string]interface{}{
						name:        "nameValue",
						description: "descValue",
						value:       "a string value",
					},
					map[string]interface{}{
						name:        "nameValue2",
						description: "descValue2",
						value:       "another string value",
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
				variation_type: "boolean",
				variations: []interface{}{
					map[string]interface{}{
						value: "true",
					},
					map[string]interface{}{
						value: "false",
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
				variation_type: "number",
				variations: []interface{}{
					map[string]interface{}{
						value: 32.5,
					},
					map[string]interface{}{
						value: 12,
					},
					map[string]interface{}{
						value: 0,
					},
				}},
			expected: []ldapi.Variation{
				{Value: ptr(float64(32.5))},
				{Value: ptr(float64(12))},
				{Value: ptr(float64(0))},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resourceData := schema.TestResourceDataRaw(t,
				map[string]*schema.Schema{variation_type: variationTypeSchema(), variations: variationsSchema()},
				tc.vars,
			)

			actualVariations, err := variationsFromResourceData(resourceData)
			assert.NoError(t, err)
			for idx, expected := range tc.expected {
				assert.Equal(t, expected.Name, actualVariations[idx].Name)
				assert.Equal(t, expected.Description, actualVariations[idx].Description)
				assert.Equal(t, expected.Value, actualVariations[idx].Value)
			}
		})
	}
}
