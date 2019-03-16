package launchdarkly

import (
	"testing"

	ldapi "github.com/launchdarkly/api-client-go"
	"github.com/stretchr/testify/require"
)

func TestEnvironmentPostFromResourceData(t *testing.T) {
	testCases := [...]struct {
		name     string
		input    map[string]interface{}
		expected ldapi.EnvironmentPost
	}{
		{
			"all fields",
			map[string]interface{}{
				name:        "envName",
				key:         "envKey",
				color:       "000000",
				default_ttl: 100.0,
			},
			ldapi.EnvironmentPost{
				Name:       "envName",
				Key:        "envKey",
				Color:      "000000",
				DefaultTtl: 100.0,
			},
		},
		{
			"all required fields",
			map[string]interface{}{
				name:  "envName",
				key:   "envKey",
				color: "000000",
			},
			ldapi.EnvironmentPost{
				Name:  "envName",
				Key:   "envKey",
				Color: "000000",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := environmentPostFromResourceData(tc.input)
			require.Equal(t, tc.expected, actual)
		})
	}

}
