package launchdarkly

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v17"
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
		{
			name: "Unsorted values get sorted",
			customProperties: map[string]interface{}{
				CUSTOM_PROPERTIES: []interface{}{
					map[string]interface{}{
						KEY:   "cp3",
						NAME:  "nameValue3",
						VALUE: []interface{}{"zebra", "apple", "banana"},
					},
				},
			},
			expected: map[string]ldapi.CustomProperty{"cp3": {
				Name:  "nameValue3",
				Value: []string{"apple", "banana", "zebra"}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resourceData := schema.TestResourceDataRaw(t,
				map[string]*schema.Schema{CUSTOM_PROPERTIES: customPropertiesSchema(false)},
				tc.customProperties,
			)

			actual := customPropertiesFromResourceData(resourceData)
			require.Equal(t, tc.expected, actual)

			actualRaw := customPropertiesToResourceData(actual)
			// After round trip, values should be sorted
			expectedAfterRoundTrip := tc.customProperties[CUSTOM_PROPERTIES].([]interface{})
			require.Len(t, actualRaw, len(expectedAfterRoundTrip))
			for i, item := range actualRaw {
				actualMap := item.(map[string]interface{})
				expectedMap := expectedAfterRoundTrip[i].(map[string]interface{})
				require.Equal(t, expectedMap[KEY], actualMap[KEY])
				require.Equal(t, expectedMap[NAME], actualMap[NAME])
				// Values should be sorted after round trip
				actualValues := actualMap[VALUE].([]interface{})
				expectedValues := expectedMap[VALUE].([]interface{})
				require.Equal(t, len(expectedValues), len(actualValues))
				// Check that values are sorted
				for j := 0; j < len(actualValues)-1; j++ {
					require.LessOrEqual(t, actualValues[j].(string), actualValues[j+1].(string), "values should be sorted")
				}
			}
		})
	}
}

func TestCustomPropertyHash(t *testing.T) {
	testCases := []struct {
		name        string
		prop1       map[string]interface{}
		prop2       map[string]interface{}
		shouldEqual bool
	}{
		{
			name: "identical properties have same hash",
			prop1: map[string]interface{}{
				KEY:   "test.key",
				NAME:  "Test Name",
				VALUE: []interface{}{"value1", "value2"},
			},
			prop2: map[string]interface{}{
				KEY:   "test.key",
				NAME:  "Test Name",
				VALUE: []interface{}{"value1", "value2"},
			},
			shouldEqual: true,
		},
		{
			name: "different keys have different hashes",
			prop1: map[string]interface{}{
				KEY:   "test.key1",
				NAME:  "Test Name",
				VALUE: []interface{}{"value1"},
			},
			prop2: map[string]interface{}{
				KEY:   "test.key2",
				NAME:  "Test Name",
				VALUE: []interface{}{"value1"},
			},
			shouldEqual: false,
		},
		{
			name: "different names have different hashes",
			prop1: map[string]interface{}{
				KEY:   "test.key",
				NAME:  "Test Name 1",
				VALUE: []interface{}{"value1"},
			},
			prop2: map[string]interface{}{
				KEY:   "test.key",
				NAME:  "Test Name 2",
				VALUE: []interface{}{"value1"},
			},
			shouldEqual: false,
		},
		{
			name: "different values have different hashes",
			prop1: map[string]interface{}{
				KEY:   "test.key",
				NAME:  "Test Name",
				VALUE: []interface{}{"value1"},
			},
			prop2: map[string]interface{}{
				KEY:   "test.key",
				NAME:  "Test Name",
				VALUE: []interface{}{"value2"},
			},
			shouldEqual: false,
		},
		{
			name: "sorted values produce same hash regardless of input order",
			prop1: map[string]interface{}{
				KEY:   "test.key",
				NAME:  "Test Name",
				VALUE: []interface{}{"zebra", "apple", "banana"},
			},
			prop2: map[string]interface{}{
				KEY:   "test.key",
				NAME:  "Test Name",
				VALUE: []interface{}{"apple", "banana", "zebra"},
			},
			shouldEqual: true,
		},
		{
			name: "different value counts have different hashes",
			prop1: map[string]interface{}{
				KEY:   "test.key",
				NAME:  "Test Name",
				VALUE: []interface{}{"value1", "value2"},
			},
			prop2: map[string]interface{}{
				KEY:   "test.key",
				NAME:  "Test Name",
				VALUE: []interface{}{"value1"},
			},
			shouldEqual: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hash1 := customPropertyHash(tc.prop1)
			hash2 := customPropertyHash(tc.prop2)

			if tc.shouldEqual {
				require.Equal(t, hash1, hash2, "hashes should be equal")
			} else {
				require.NotEqual(t, hash1, hash2, "hashes should be different")
			}
		})
	}
}
