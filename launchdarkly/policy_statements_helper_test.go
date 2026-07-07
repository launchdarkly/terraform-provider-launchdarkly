package launchdarkly

import (
	"encoding/json"
	"strings"
	"testing"

	ldapi "github.com/launchdarkly/api-client-go/v23"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicyStatementsFromJSON_RoundTrip(t *testing.T) {
	input := `[
        {
            "effect": "allow",
            "resources": ["proj/*:env/staging"],
            "actions": ["*"]
        },
        {
            "effect": "deny",
            "not_resources": ["proj/*:env/production"],
            "actions": ["updateOn"]
        }
    ]`

	statements, err := policyStatementsFromJSON(input)
	require.NoError(t, err)
	require.Len(t, statements, 2)

	assert.Equal(t, "allow", statements[0].Effect)
	assert.Equal(t, []string{"proj/*:env/staging"}, statements[0].Resources)
	assert.Equal(t, []string{"*"}, statements[0].Actions)
	assert.Nil(t, statements[0].NotResources)
	assert.Nil(t, statements[0].NotActions)

	assert.Equal(t, "deny", statements[1].Effect)
	assert.Equal(t, []string{"proj/*:env/production"}, statements[1].NotResources)
	assert.Equal(t, []string{"updateOn"}, statements[1].Actions)
	assert.Nil(t, statements[1].Resources)
	assert.Nil(t, statements[1].NotActions)

	encoded, err := policyStatementsToJSON(statementPostsToStatementReps(statements))
	require.NoError(t, err)
	var roundtrip []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(encoded), &roundtrip))
	require.Len(t, roundtrip, 2)
	assert.Equal(t, "allow", roundtrip[0]["effect"])
	assert.Equal(t, "deny", roundtrip[1]["effect"])
}

func TestPolicyStatementsFromJSON_EmptyReturnsNil(t *testing.T) {
	statements, err := policyStatementsFromJSON("")
	require.NoError(t, err)
	assert.Nil(t, statements)

	statements, err = policyStatementsFromJSON("   \n   ")
	require.NoError(t, err)
	assert.Nil(t, statements)
}

func TestPolicyStatementsFromJSON_RejectsInvalid(t *testing.T) {
	cases := map[string]struct {
		input  string
		errSub string
	}{
		"not JSON": {
			input:  `not-json`,
			errSub: "invalid JSON",
		},
		"object not array": {
			input:  `{"effect":"allow"}`,
			errSub: "invalid JSON",
		},
		"missing effect": {
			input:  `[{"resources":["proj/*:env/staging"],"actions":["*"]}]`,
			errSub: "'effect' is required",
		},
		"effect not string": {
			input:  `[{"effect":1,"resources":["proj/*:env/staging"],"actions":["*"]}]`,
			errSub: "must be a string",
		},
		"both resources and not_resources": {
			input:  `[{"effect":"allow","resources":["a"],"not_resources":["b"],"actions":["*"]}]`,
			errSub: "cannot contain both 'resources' and 'not_resources'",
		},
		"missing resources and not_resources": {
			input:  `[{"effect":"allow","actions":["*"]}]`,
			errSub: "must contain either 'resources' or 'not_resources'",
		},
		"both actions and not_actions": {
			input:  `[{"effect":"allow","resources":["a"],"actions":["*"],"not_actions":["b"]}]`,
			errSub: "cannot contain both 'actions' and 'not_actions'",
		},
		"missing actions and not_actions": {
			input:  `[{"effect":"allow","resources":["a"]}]`,
			errSub: "must contain either 'actions' or 'not_actions'",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := policyStatementsFromJSON(tc.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errSub)
		})
	}
}

func TestPolicyStatementsToJSON_EmptyReturnsEmptyString(t *testing.T) {
	encoded, err := policyStatementsToJSON(nil)
	require.NoError(t, err)
	assert.Equal(t, "", encoded)
}

func TestPolicyStatementsToJSON_OmitsEmptySliceFields(t *testing.T) {
	input := `[{"effect":"allow","resources":["proj/*"],"actions":["*"]}]`
	statements, err := policyStatementsFromJSON(input)
	require.NoError(t, err)

	encoded, err := policyStatementsToJSON(statementPostsToStatementReps(statements))
	require.NoError(t, err)

	assert.NotContains(t, encoded, "not_resources")
	assert.NotContains(t, encoded, "not_actions")
}

func TestPolicyStatementsFromJSON_TolerantOfTrimmableWhitespace(t *testing.T) {
	input := "\n  [{\"effect\":\"allow\",\"resources\":[\"proj/*\"],\"actions\":[\"*\"]}]\n  "
	statements, err := policyStatementsFromJSON(input)
	require.NoError(t, err)
	require.Len(t, statements, 1)
	assert.True(t, strings.HasSuffix(input, "  "))
}

// TestPolicyStatementsFromJSON_StatementShapeMatchesBlockForm ensures the
// JSON path produces ldapi.StatementPost values matching the block form's
// frameworkPolicyStatementsFromList output for an equivalent input.
func TestPolicyStatementsFromJSON_StatementShapeMatchesBlockForm(t *testing.T) {
	jsonStatements, err := policyStatementsFromJSON(
		`[{"effect":"allow","resources":["proj/*"],"actions":["*"]}]`,
	)
	require.NoError(t, err)
	require.Len(t, jsonStatements, 1)

	blockStatement := frameworkPolicyStatementModel{
		Resources: []string{"proj/*"},
		Actions:   []string{"*"},
		Effect:    "allow",
	}
	blockStmtPost, blockDiags := blockStatement.toLDAPI()
	require.False(t, blockDiags.HasError(), blockDiags)

	assert.Equal(t, []ldapi.StatementPost{blockStmtPost}, jsonStatements)
}
