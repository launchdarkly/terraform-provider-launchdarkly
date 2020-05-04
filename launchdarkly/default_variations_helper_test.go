package launchdarkly

import (
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultVariationsFromResourceData(t *testing.T) {
	testCases := []struct {
		name        string
		vars        map[string]interface{}
		expected    *ldapi.Defaults
		expectedErr error
	}{
		{
			name: "no defaults",
			vars: map[string]interface{}{
				VARIATION_TYPE: "boolean",
				VARIATIONS: []interface{}{
					map[string]interface{}{
						VALUE: "true",
					},
					map[string]interface{}{
						VALUE: "false",
					},
				},
			},
			expected: nil,
		},
		{
			name: "basic defaults, implicit variations",
			vars: map[string]interface{}{
				VARIATION_TYPE:        "boolean",
				DEFAULT_ON_VARIATION:  "true",
				DEFAULT_OFF_VARIATION: "false",
			},
			expected: &ldapi.Defaults{
				OnVariation:  0,
				OffVariation: 1,
			},
		},
		{
			name: "invalid defaults, implicit variations",
			vars: map[string]interface{}{
				VARIATION_TYPE:        "boolean",
				DEFAULT_ON_VARIATION:  "a",
				DEFAULT_OFF_VARIATION: "c",
			},
			expectedErr: errors.New(`default_on_variation "a" is not defined as a variation`),
		},
		{
			name: "basic defaults",
			vars: map[string]interface{}{
				VARIATION_TYPE: "boolean",
				VARIATIONS: []interface{}{
					map[string]interface{}{
						VALUE: "true",
					},
					map[string]interface{}{
						VALUE: "false",
					},
				},
				DEFAULT_ON_VARIATION:  "true",
				DEFAULT_OFF_VARIATION: "false",
			},
			expected: &ldapi.Defaults{
				OnVariation:  0,
				OffVariation: 1,
			},
		},
		{
			name: "invalid default on value",
			vars: map[string]interface{}{
				VARIATION_TYPE: "boolean",
				VARIATIONS: []interface{}{
					map[string]interface{}{
						VALUE: "true",
					},
					map[string]interface{}{
						VALUE: "false",
					},
				},
				DEFAULT_ON_VARIATION:  "not a boolean",
				DEFAULT_OFF_VARIATION: "false",
			},
			expectedErr: errors.New(`default_on_variation "not a boolean" is not defined as a variation`),
		},
		{
			name: "invalid default off value",
			vars: map[string]interface{}{
				VARIATION_TYPE: "boolean",
				VARIATIONS: []interface{}{
					map[string]interface{}{
						VALUE: "true",
					},
					map[string]interface{}{
						VALUE: "false",
					},
				},
				DEFAULT_ON_VARIATION:  "true",
				DEFAULT_OFF_VARIATION: "not a boolean",
			},
			expectedErr: errors.New(`default_off_variation "not a boolean" is not defined as a variation`),
		},
		{
			name: "missing default off",
			vars: map[string]interface{}{
				VARIATION_TYPE: "boolean",
				VARIATIONS: []interface{}{
					map[string]interface{}{
						VALUE: "true",
					},
					map[string]interface{}{
						VALUE: "false",
					},
				},
				DEFAULT_ON_VARIATION: "true",
			},
			expectedErr: errors.New(`default_off_variation is required when default_on_variation is defined`),
		},
		{
			name: "missing default on",
			vars: map[string]interface{}{
				VARIATION_TYPE: "boolean",
				VARIATIONS: []interface{}{
					map[string]interface{}{
						VALUE: "true",
					},
					map[string]interface{}{
						VALUE: "false",
					},
				},
				DEFAULT_OFF_VARIATION: "false",
			},
			expectedErr: errors.New(`default_on_variation is required when default_off_variation is defined`),
		},
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
				},
				DEFAULT_ON_VARIATION:  "a string value",
				DEFAULT_OFF_VARIATION: "a string value",
			},
			expected: &ldapi.Defaults{
				OnVariation:  0,
				OffVariation: 0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resourceData := schema.TestResourceDataRaw(t,
				map[string]*schema.Schema{VARIATION_TYPE: variationTypeSchema(), VARIATIONS: variationsSchema(),
					DEFAULT_ON_VARIATION: {
						Type:     schema.TypeString,
						Optional: true,
					},
					DEFAULT_OFF_VARIATION: {
						Type:     schema.TypeString,
						Optional: true,
					}},
				tc.vars,
			)

			actual, err := defaultVariationsFromResourceData(resourceData)
			require.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
