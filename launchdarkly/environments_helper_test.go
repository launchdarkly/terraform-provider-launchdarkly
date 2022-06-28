package launchdarkly

import (
	"testing"

	ldapi "github.com/launchdarkly/api-client-go/v10"
	"github.com/stretchr/testify/assert"
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
				NAME:        "envName",
				KEY:         "envKey",
				COLOR:       "000000",
				DEFAULT_TTL: 50,
			},
			ldapi.EnvironmentPost{
				Name:       "envName",
				Key:        "envKey",
				Color:      "000000",
				DefaultTtl: ldapi.PtrInt32(50),
			},
		},
		{
			"all required fields",
			map[string]interface{}{
				NAME:  "envName",
				KEY:   "envKey",
				COLOR: "000000",
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

func TestEnvironmentToResourceData(t *testing.T) {
	testCases := []struct {
		name     string
		input    ldapi.Environment
		expected envResourceData
	}{
		{
			name: "standard environment",
			input: ldapi.Environment{
				Key:                "test-env",
				Name:               "Test Env",
				ApiKey:             "sdk-234123123",
				MobileKey:          "b1235363456",
				Id:                 "b234234234234",
				Color:              "FFFFFF",
				DefaultTtl:         60,
				SecureMode:         true,
				DefaultTrackEvents: true,
				RequireComments:    true,
				ConfirmChanges:     true,
				Tags:               []string{"test"},
				ApprovalSettings: &ldapi.ApprovalSettings{
					Required:                true,
					MinNumApprovals:         3,
					CanApplyDeclinedChanges: true,
					RequiredApprovalTags:    []string{"approval"},
					CanReviewOwnRequest:     true,
				},
			},
			expected: envResourceData{
				KEY:                  "test-env",
				NAME:                 "Test Env",
				API_KEY:              "sdk-234123123",
				MOBILE_KEY:           "b1235363456",
				CLIENT_SIDE_ID:       "b234234234234",
				COLOR:                "FFFFFF",
				DEFAULT_TTL:          60,
				SECURE_MODE:          true,
				DEFAULT_TRACK_EVENTS: true,
				REQUIRE_COMMENTS:     true,
				CONFIRM_CHANGES:      true,
				TAGS:                 []string{"test"},
				APPROVAL_SETTINGS: []map[string]interface{}{
					{
						CAN_REVIEW_OWN_REQUEST:     true,
						MIN_NUM_APPROVALS:          int32(3),
						CAN_APPLY_DECLINED_CHANGES: true,
						REQUIRED_APPROVAL_TAGS:     []string{"approval"},
						REQUIRED:                   true,
					},
				},
			},
		},
		{
			name: "without approval settings",
			input: ldapi.Environment{
				Key:                "test-env",
				Name:               "Test Env",
				ApiKey:             "sdk-234123123",
				MobileKey:          "b1235363456",
				Id:                 "b234234234234",
				Color:              "FFFFFF",
				DefaultTtl:         60,
				SecureMode:         true,
				DefaultTrackEvents: true,
				RequireComments:    true,
				ConfirmChanges:     true,
				Tags:               []string{"test"},
			},
			expected: envResourceData{
				KEY:                  "test-env",
				NAME:                 "Test Env",
				API_KEY:              "sdk-234123123",
				MOBILE_KEY:           "b1235363456",
				CLIENT_SIDE_ID:       "b234234234234",
				COLOR:                "FFFFFF",
				DEFAULT_TTL:          60,
				SECURE_MODE:          true,
				DEFAULT_TRACK_EVENTS: true,
				REQUIRE_COMMENTS:     true,
				CONFIRM_CHANGES:      true,
				TAGS:                 []string{"test"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := environmentToResourceData(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
