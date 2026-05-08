package launchdarkly

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"
)

func TestIsOmittedEmbeddedSchemaAttrErr_wrapped(t *testing.T) {
	inner := fmt.Errorf("SetNew: invalid key: include_in_snippet")
	wrapped := fmt.Errorf("cannot compute the instance diff: %w", inner)
	if !isOmittedEmbeddedSchemaAttrErr(wrapped, "include_in_snippet") {
		t.Fatal("expected wrapped invalid key error to match")
	}
}

func TestIsOmittedEmbeddedSchemaAttrErr(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		attr   string
		wantOK bool
	}{
		{
			name:   "SetNew invalid key",
			err:    fmt.Errorf("SetNew: invalid key: include_in_snippet"),
			attr:   "include_in_snippet",
			wantOK: true,
		},
		{
			name:   "Set invalid key prefix",
			err:    fmt.Errorf("Set: invalid key: default_client_side_availability"),
			attr:   "default_client_side_availability",
			wantOK: true,
		},
		{
			name:   "wrong attr",
			err:    fmt.Errorf("SetNew: invalid key: other"),
			attr:   "include_in_snippet",
			wantOK: false,
		},
		{
			name:   "Invalid address",
			err:    errors.New(`Invalid address to set: []string{"include_in_snippet"}`),
			attr:   "include_in_snippet",
			wantOK: true,
		},
		{
			name:   "nil",
			err:    nil,
			attr:   "include_in_snippet",
			wantOK: false,
		},
		{
			name:   "empty attr",
			err:    fmt.Errorf("Set: invalid key: x"),
			attr:   "",
			wantOK: false,
		},
		{
			name:   "false positive: validation message mentioning the attr",
			err:    errors.New("validation failed: invalid value for include_in_snippet"),
			attr:   "include_in_snippet",
			wantOK: false,
		},
		{
			name:   "false positive: bare 'invalid key' without colon prefix",
			err:    errors.New("invalid key include_in_snippet"),
			attr:   "include_in_snippet",
			wantOK: false,
		},
		{
			name:   "false positive: address-shape with different attr",
			err:    errors.New(`Invalid address to set: []string{"other"}`),
			attr:   "include_in_snippet",
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isOmittedEmbeddedSchemaAttrErr(tt.err, tt.attr); got != tt.wantOK {
				t.Fatalf("isOmittedEmbeddedSchemaAttrErr(...) = %v, want %v", got, tt.wantOK)
			}
		})
	}
}

// resourceDataSetSkipMissingKey must propagate non-suppression errors verbatim so callers do not
// silently lose real type-mismatch / coercion failures.
func TestResourceDataSetSkipMissingKey_propagatesUnrelatedErrors(t *testing.T) {
	t.Parallel()

	d := schema.TestResourceDataRaw(t, map[string]*schema.Schema{
		"some_int": {Type: schema.TypeInt, Optional: true},
	}, map[string]interface{}{})

	err := resourceDataSetSkipMissingKey(d, "some_int", "not-an-int")
	require.Error(t, err, "type-coercion errors must not be swallowed by the missing-key shim")
}

// resourceDataSetSkipMissingKey must swallow exactly the SDK 'Invalid address to set' error when
// the attribute does not exist in the schema.
func TestResourceDataSetSkipMissingKey_suppressesMissingKey(t *testing.T) {
	t.Parallel()

	d := schema.TestResourceDataRaw(t, map[string]*schema.Schema{
		"present": {Type: schema.TypeString, Optional: true},
	}, map[string]interface{}{})

	err := resourceDataSetSkipMissingKey(d, "absent", "value")
	require.NoError(t, err, "missing schema key must be suppressed")
}
