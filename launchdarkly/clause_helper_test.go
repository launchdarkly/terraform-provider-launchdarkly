package launchdarkly

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClauseFromResourceData(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		clause            map[string]interface{}
		expected          ldapi.Clause
		expectedErrString *string
	}{
		{
			name: "string clause values",
			clause: map[string]interface{}{
				ATTRIBUTE: "country",
				OP:        "startsWith",
				VALUES: []interface{}{
					"en",
					"gr",
				},
				NEGATE: false,
			},
			expected: ldapi.Clause{
				Attribute: "country",
				Op:        "startsWith",
				Values:    []interface{}{"en", "gr"},
				Negate:    false,
			},
		},
		{
			name: "number clause values",
			clause: map[string]interface{}{
				ATTRIBUTE:  "answer",
				OP:         "greaterThan",
				VALUES:     []interface{}{"187"},
				VALUE_TYPE: NUMBER_CLAUSE_VALUE,
				NEGATE:     false,
			},
			expected: ldapi.Clause{
				Attribute: "answer",
				Op:        "greaterThan",
				Values:    []interface{}{float64(187)},
				Negate:    false,
			},
		},
		{
			name: "boolean clause values",
			clause: map[string]interface{}{
				ATTRIBUTE:  "is_vip",
				OP:         "in",
				VALUES:     []interface{}{"true"},
				VALUE_TYPE: BOOL_CLAUSE_VALUE,
				NEGATE:     false,
			},
			expected: ldapi.Clause{
				Attribute: "is_vip",
				Op:        "in",
				Values:    []interface{}{true},
				Negate:    false,
			},
		},
		{
			name: "string clause values with ambiguous values",
			clause: map[string]interface{}{
				ATTRIBUTE:  "test",
				OP:         "in",
				VALUES:     []interface{}{"true", "42.8", "wow"},
				VALUE_TYPE: STRING_CLAUSE_VALUE,
				NEGATE:     false,
			},
			expected: ldapi.Clause{
				Attribute: "test",
				Op:        "in",
				Values:    []interface{}{"true", "42.8", "wow"},
				Negate:    false,
			},
		},
		{
			name: "invalid boolean value throws an error",
			clause: map[string]interface{}{
				ATTRIBUTE:  "test",
				OP:         "in",
				VALUES:     []interface{}{"wow"},
				VALUE_TYPE: BOOL_CLAUSE_VALUE,
				NEGATE:     false,
			},
			expectedErrString: strPtr(`invalid boolean string "wow"`),
		},
		{
			name: "invalid number value throws an error",
			clause: map[string]interface{}{
				ATTRIBUTE:  "test",
				OP:         "in",
				VALUES:     []interface{}{"wow"},
				VALUE_TYPE: NUMBER_CLAUSE_VALUE,
				NEGATE:     false,
			},
			expectedErrString: strPtr(`invalid number string "wow"`),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resourceData := schema.TestResourceDataRaw(
				t,
				map[string]*schema.Schema{CLAUSES: clauseSchema()},
				map[string]interface{}{
					CLAUSES: []interface{}{tc.clause},
				},
			)
			clauses := resourceData.Get(CLAUSES).([]interface{})

			ldClause, err := clauseFromResourceData(clauses[0])
			if tc.expectedErrString != nil {
				assert.EqualError(t, err, *tc.expectedErrString)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, ldClause)
			}
		})
	}
}

func TestClausesToResourceData(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		clauses  []ldapi.Clause
		expected []interface{}
	}{
		{
			name: "string clause values",
			clauses: []ldapi.Clause{
				{
					Attribute: "country",
					Op:        "in",
					Values:    []interface{}{"en", "gb"},
					Negate:    true,
				},
			},
			expected: []interface{}{
				map[string]interface{}{
					ATTRIBUTE:  "country",
					OP:         "in",
					VALUES:     []interface{}{"en", "gb"},
					VALUE_TYPE: STRING_CLAUSE_VALUE,
					NEGATE:     true,
				},
			},
		},
		{
			name: "bool clause value",
			clauses: []ldapi.Clause{
				{
					Attribute: "is_vip",
					Op:        "in",
					Values:    []interface{}{false},
					Negate:    true,
				},
			},
			expected: []interface{}{
				map[string]interface{}{
					ATTRIBUTE:  "is_vip",
					OP:         "in",
					VALUES:     []interface{}{"false"},
					VALUE_TYPE: BOOL_CLAUSE_VALUE,
					NEGATE:     true,
				},
			},
		},
		{
			name: "number clause value",
			clauses: []ldapi.Clause{
				{
					Attribute: "answer",
					Op:        "in",
					Values:    []interface{}{float64(42)},
					Negate:    false,
				},
			},
			expected: []interface{}{
				map[string]interface{}{
					ATTRIBUTE:  "answer",
					OP:         "in",
					VALUES:     []interface{}{"42"},
					VALUE_TYPE: NUMBER_CLAUSE_VALUE,
					NEGATE:     false,
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual, err := clausesToResourceData(tc.clauses)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
