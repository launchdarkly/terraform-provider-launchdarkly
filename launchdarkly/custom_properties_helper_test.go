package launchdarkly

import (
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/launchdarkly/api-client-go"
	"github.com/stretchr/testify/require"
)

func TestCustomPropertiesFromResourceData(t *testing.T) {
	testCases := []struct {
		name             string
		customProperties map[string]interface{}
		expected         map[string]ldapi.CustomProperty
	}{
		{
			name: "basic custom property",
			customProperties: map[string]interface{}{
				custom_properties: []map[string]interface{}{
					{key: "cp1",
						name:  "nameValue",
						value: []interface{}{"a cp value"},
					},
				},
			},
			expected: map[string]ldapi.CustomProperty{"cp1": {"nameValue", []string{"a cp value"}}},
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
		})
	}
}
