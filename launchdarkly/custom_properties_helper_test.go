package launchdarkly

import (
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/launchdarkly/api-client-go"
	"github.com/stretchr/testify/require"
)

func TestCustomPropertiesRoundTripConversion(t *testing.T) {
	testCases := []struct {
		name             string
		customProperties map[string]interface{}
		expected         map[string]ldapi.CustomProperty
	}{
		{
			name: "basic custom property",
			customProperties: map[string]interface{}{
				custom_properties: []map[string]interface{}{
					{
						key:   "cp1",
						name:  "nameValue",
						value: []string{"a cp value"},
					},
				},
			},
			expected: map[string]ldapi.CustomProperty{"cp1": {
				Name:  "nameValue",
				Value: []string{"a cp value"}},
			},
		},
		{
			name: "Multiple custom properties",
			customProperties: map[string]interface{}{
				custom_properties: []map[string]interface{}{
					{
						key:   "cp2",
						name:  "nameValue2",
						value: []string{"a cp value1", "a cp value2", "a cp value3"},
					},
				},
			},
			expected: map[string]ldapi.CustomProperty{"cp2": {
				Name:  "nameValue2",
				Value: []string{"a cp value1", "a cp value2", "a cp value3"}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resourceData := schema.TestResourceDataRaw(t,
				map[string]*schema.Schema{custom_properties: customPropertiesSchema()},
				tc.customProperties,
			)

			actual := customPropertiesFromResourceData(resourceData)
			require.Equal(t, tc.expected, actual)

			actualRaw := customPropertiesToResourceData(actual)
			require.Equal(t, tc.customProperties[custom_properties], actualRaw)

		})
	}
}
