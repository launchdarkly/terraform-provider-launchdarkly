package launchdarkly

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateJsonSchemaStringFunc(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		value     interface{}
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "valid schema",
			value:   `{"type":"object","properties":{"name":{"type":"string"}}}`,
			wantErr: false,
		},
		{
			name:      "invalid json",
			value:     `{"type":"object"`,
			wantErr:   true,
			errSubstr: "invalid JSON",
		},
		{
			name:      "invalid schema structure",
			value:     `{"type":1}`,
			wantErr:   true,
			errSubstr: "invalid JSON Schema",
		},
		{
			name:    "empty string",
			value:   "",
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, errs := validateJsonSchemaStringFunc(tc.value, "schema_json")
			if !tc.wantErr {
				require.Empty(t, errs)
				return
			}

			require.NotEmpty(t, errs)
			assert.ErrorContains(t, errs[0], tc.errSubstr)
		})
	}
}
