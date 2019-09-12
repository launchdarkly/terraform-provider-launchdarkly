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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resourceData := schema.TestResourceDataRaw(t,
				map[string]*schema.Schema{variation_type: variationTypeSchema(), variations: variationsSchema()},
				tc.vars,
			)

			actualVariations := variationsFromResourceData(resourceData)
			assert.ElementsMatch(t, tc.expected, actualVariations)
		})
	}
}
