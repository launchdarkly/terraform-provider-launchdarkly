package launchdarkly

import (
	"github.com/pkg/errors"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/launchdarkly/api-client-go"
	"github.com/stretchr/testify/require"
)

func TestVariationsFromResourceData(t *testing.T) {
	testCases := []struct {
		name          string
		variationType string
		variations    map[string]interface{}
		expected      []ldapi.Variation
		expectedError error
	}{
		{
			name:          "string variations",
			variationType: "string",
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
			expectedError: nil,
		},
		{
			name:          "float variations",
			variationType: "float",
			variations: map[string]interface{}{
				variations: []map[string]interface{}{
					{
						name:        "nameValue",
						description: "descValue",
						value:       "10000.0112",
					},
					{
						name:        "nameValue2",
						description: "descValue2",
						value:       "0.231",
					},
				}},
			expected: []ldapi.Variation{
				{Name: "nameValue", Description: "descValue", Value: ptr(10000.0112)},
				{Name: "nameValue2", Description: "descValue2", Value: ptr(0.231)},
			},
			expectedError: nil,
		},
		{
			name:          "json variations",
			variationType: "json",
			variations: map[string]interface{}{
				variations: []map[string]interface{}{
					{
						name:        "nameValue",
						description: "descValue",
						value:       `{"key1":"value1"}`,
					},
					{
						name:        "nameValue2",
						description: "descValue2",
						value:       `{"key1":"value2"}`,
					},
				}},
			expected: []ldapi.Variation{
				{Name: "nameValue", Description: "descValue", Value: ptr(map[string]interface{}{"key1": "value1"})},
				{Name: "nameValue2", Description: "descValue2", Value: ptr(map[string]interface{}{"key1": "value2"})},
			},
			expectedError: nil,
		},
		{
			name:          "unparsable float variation",
			variationType: "float",
			variations: map[string]interface{}{
				variations: []map[string]interface{}{
					{
						name:        "nameValue",
						description: "descValue",
						value:       "not a float",
					},
				}},
			expected:      []ldapi.Variation{},
			expectedError: errors.New("Expected string: \"not a float\" to parse as a float.: strconv.ParseFloat: parsing \"not a float\": invalid syntax"),
		},
		{
			name:          "invalid json",
			variationType: "json",
			variations: map[string]interface{}{
				variations: []map[string]interface{}{
					{
						name:        "nameValue",
						description: "descValue",
						value:       "not actual json",
					},
				}},
			expected:      []ldapi.Variation{},
			expectedError: errors.New("Expected string: \"not actual json\" to be valid json.: invalid character 'o' in literal null (expecting 'u')"),
		},
		{
			name:          "unknown variation type",
			variationType: "yaml",
			variations: map[string]interface{}{
				variations: []map[string]interface{}{
					{value: `some yaml}`},
					{value: `some more yaml`},
				}},
			expected:      []ldapi.Variation{},
			expectedError: errors.New("unexpected variation type: \"yaml\" must be one of: string,float,json"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resourceData := schema.TestResourceDataRaw(t,
				map[string]*schema.Schema{variations: variationsSchema()},
				tc.variations,
			)

			actualVariations, err := variationsFromResourceData(resourceData, tc.variationType)
			if tc.expectedError != nil {
				require.EqualError(t, err, tc.expectedError.Error())
			} else {
				require.NoError(t, err)
				require.ElementsMatch(t, tc.expected, actualVariations)
			}
		})
	}
}
