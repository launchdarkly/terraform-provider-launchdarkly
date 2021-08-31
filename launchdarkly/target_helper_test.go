package launchdarkly

import (
	"testing"

	ldapi "github.com/launchdarkly/api-client-go"
	"github.com/stretchr/testify/assert"
)

func TestTargetsToResourceData(t *testing.T) {
	testCases := []struct {
		name     string
		targets  []ldapi.Target
		expected []interface{}
	}{
		{
			name: "standard",
			targets: []ldapi.Target{
				{
					Values:    []string{"test1"},
					Variation: 0,
				},
				{
					Values:    []string{"test2"},
					Variation: 1,
				},
			},
			expected: []interface{}{
				map[string]interface{}{
					"values":    []string{"test1"},
					"variation": 0,
				},
				map[string]interface{}{
					"values":    []string{"test2"},
					"variation": 1,
				},
			},
		},
		{
			name: "out of order",
			targets: []ldapi.Target{
				{
					Values:    []string{"test1"},
					Variation: 1,
				},
				{
					Values:    []string{"test2"},
					Variation: 0,
				},
			},
			expected: []interface{}{
				map[string]interface{}{
					"values":    []string{"test1"},
					"variation": 1,
				},
				map[string]interface{}{
					"values":    []string{"test2"},
					"variation": 0,
				},
			},
		},
		{
			name: "missing variation 0",
			targets: []ldapi.Target{
				{
					Values:    []string{"test2"},
					Variation: 1,
				},
			},
			expected: []interface{}{
				map[string]interface{}{
					"values":    []string{"test2"},
					"variation": 1,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, targetsToResourceData(tc.targets))
		})
	}
}
