package launchdarkly

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v7"
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
				CUSTOM_PROPERTIES: []interface{}{
					map[string]interface{}{
						KEY:   "cp1",
						NAME:  "nameValue",
						VALUE: []interface{}{"a cp value"},
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
				CUSTOM_PROPERTIES: []interface{}{
					map[string]interface{}{
						KEY:   "cp2",
						NAME:  "nameValue2",
						VALUE: []interface{}{"a cp value1", "a cp value2", "a cp value3"},
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
				map[string]*schema.Schema{CUSTOM_PROPERTIES: customPropertiesSchema()},
				tc.customProperties,
			)

			actual := customPropertiesFromResourceData(resourceData)
			require.Equal(t, tc.expected, actual)

			actualRaw := customPropertiesToResourceData(actual)
			require.Equal(t, tc.customProperties[CUSTOM_PROPERTIES], actualRaw)
		})
	}
}
