package launchdarkly

import (
	"testing"

	ldapi "github.com/launchdarkly/api-client-go"
	"github.com/stretchr/testify/assert"

	"github.com/hashicorp/terraform/helper/schema"
)

func TestVariationsFromResourceData(t *testing.T) {
	testCases := []struct {
		name       string
		variations map[string]interface{}
		expected   []ldapi.Variation
	}{
		{
			name: "string variations",
			variations: map[string]interface{}{
				variations: []map[string]interface{}{
					{
						name:        "nameValue",
						description: "descValue",
						value:       "a string value",
					},
					{
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
			variations: map[string]interface{}{
				variations: []map[string]interface{}{
					{
						value: "true",
					},
					{
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
				map[string]*schema.Schema{variations: variationsSchema()},
				tc.variations,
			)

			actualVariations := variationsFromResourceData(resourceData)
			assert.ElementsMatch(t, tc.expected, actualVariations)
		})
	}
}
