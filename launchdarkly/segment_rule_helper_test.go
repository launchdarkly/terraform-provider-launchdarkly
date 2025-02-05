package launchdarkly

import (
	"testing"

	ldapi "github.com/launchdarkly/api-client-go/v17"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSegmentRuleFromResourceData(t *testing.T) {
	cases := []struct {
		name     string
		input    map[string]interface{}
		expected ldapi.UserSegmentRule
	}{
		{
			// https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/79
			name: "Zero case should not create a UserSegmentRule with 0 weight",
			input: map[string]interface{}{
				WEIGHT:    0,
				BUCKET_BY: "",
				CLAUSES:   []interface{}{},
			},
			expected: ldapi.UserSegmentRule{
				Clauses: []ldapi.Clause{},
			},
		},
		{
			name: "Clauses only - most typical case",
			input: map[string]interface{}{
				WEIGHT:    0,
				BUCKET_BY: "",
				CLAUSES: []interface{}{
					map[string]interface{}{
						ATTRIBUTE:  "country",
						OP:         "in",
						NEGATE:     false,
						VALUES:     []interface{}{"us", "gb"},
						VALUE_TYPE: "string",
					},
				},
			},
			expected: ldapi.UserSegmentRule{
				Clauses: []ldapi.Clause{
					{
						Attribute: "country",
						Op:        "in",
						Negate:    false,
						Values:    []interface{}{"us", "gb"},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := segmentRuleFromResourceData(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
