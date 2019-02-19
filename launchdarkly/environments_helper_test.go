package launchdarkly

import (
	"testing"

	"github.com/launchdarkly/api-client-go"
	"github.com/stretchr/testify/require"
)

//func TestEnvironmentFromResourceData(t *testing.T) {
//	resourceData := schema.TestResourceDataRaw(t,
//		environmentSchema(),
//		map[string]interface{}{
//				key:                  "testEnvKey",
//				name:                 "testEnvName",
//				color:                "ffffff",
//				default_ttl:          0.5,
//				secure_mode:          true,
//				default_track_events: true,
//				tags:                 []string{"tag1", "tag2"},
//		},
//	)
//
//	expected := []ldapi.Environment{
//		{
//			Key:                "",
//			Name:               "",
//			ApiKey:             "",
//			MobileKey:          "",
//			Color:              "",
//			DefaultTtl:         0,
//			SecureMode:         false,
//			DefaultTrackEvents: false,
//			Tags:               nil,
//		},
//	}
//
//	actual := environmentFromResourceData(resourceData)
//	require.Equal(t, expected, actual)
//}

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
