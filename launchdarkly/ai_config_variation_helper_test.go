package launchdarkly

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessagesFromResourceData(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		messages     []interface{}
		expectErr    bool
		errSubstring string
	}{
		{
			name: "valid messages",
			messages: []interface{}{
				map[string]interface{}{
					ROLE:    "user",
					CONTENT: "hello",
				},
				map[string]interface{}{
					ROLE:    "assistant",
					CONTENT: "world",
				},
			},
			expectErr: false,
		},
		{
			name: "invalid list item type",
			messages: []interface{}{
				"not-a-map",
			},
			expectErr:    true,
			errSubstring: "value type",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d := schema.TestResourceDataRaw(t, aiConfigVariationSchema(false), map[string]interface{}{
				MESSAGES: tc.messages,
			})

			got, err := messagesFromResourceData(d)
			if !tc.expectErr {
				require.NoError(t, err)
				require.Len(t, got, len(tc.messages))
				return
			}

			require.Error(t, err)
			assert.ErrorContains(t, err, tc.errSubstring)
		})
	}
}
