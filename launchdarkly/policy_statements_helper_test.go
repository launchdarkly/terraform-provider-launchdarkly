package launchdarkly

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v22"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicyStatementsRoundTripConversion(t *testing.T) {
	statementResources := []string{"proj/*"}
	statementActions := []string{"*"}
	statementPostResources1 := []string{"proj/*:env/*;qa_*"}
	statementPostResources2 := []string{"proj/*:env/*;qa_*:/flag/*"}
	statementPostActions := []string{"*"}

	testCases := []struct {
		name             string
		policyStatements map[string]interface{}
		expected         []ldapi.StatementPost
	}{
		{
			name: "basic policy statement",
			policyStatements: map[string]interface{}{
				POLICY_STATEMENTS: []interface{}{
					map[string]interface{}{
						RESOURCES: []interface{}{"proj/*"},
						ACTIONS:   []interface{}{"*"},
						EFFECT:    "allow",
					},
				},
			},
			expected: []ldapi.StatementPost{
				{
					Resources: statementResources,
					Actions:   statementActions,
					Effect:    "allow",
				},
			},
		},
		{
			name: "QA team policy",
			policyStatements: map[string]interface{}{
				POLICY_STATEMENTS: []interface{}{
					map[string]interface{}{
						RESOURCES: []interface{}{"proj/*:env/*;qa_*"},
						ACTIONS:   []interface{}{"*"},
						EFFECT:    "allow",
					},
					map[string]interface{}{
						RESOURCES: []interface{}{"proj/*:env/*;qa_*:/flag/*"},
						ACTIONS:   []interface{}{"*"},
						EFFECT:    "allow",
					},
				},
			},
			expected: []ldapi.StatementPost{
				{
					Resources: statementPostResources1,
					Actions:   statementPostActions,
					Effect:    "allow",
				},
				{
					Resources: statementPostResources2,
					Actions:   statementPostActions,
					Effect:    "allow",
				},
			},
		},
		{
			name: "not_resources example",
			policyStatements: map[string]interface{}{
				POLICY_STATEMENTS: []interface{}{
					map[string]interface{}{
						NOT_RESOURCES: []interface{}{"proj/*:env/production:flag/*"},
						ACTIONS:       []interface{}{"*"},
						EFFECT:        "allow",
					},
				},
			},
			expected: []ldapi.StatementPost{
				{
					NotResources: *strArrayPtr([]string{"proj/*:env/production:flag/*"}),
					Actions:      statementPostActions,
					Effect:       "allow",
				},
			},
		},
		{
			name: "not_actions example",
			policyStatements: map[string]interface{}{
				POLICY_STATEMENTS: []interface{}{
					map[string]interface{}{
						RESOURCES:   []interface{}{"proj/*:env/production:flag/*"},
						NOT_ACTIONS: []interface{}{"*"},
						EFFECT:      "allow",
					},
				},
			},
			expected: []ldapi.StatementPost{
				{
					Resources:  *strArrayPtr([]string{"proj/*:env/production:flag/*"}),
					NotActions: statementPostActions,
					Effect:     "allow",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			statementsData := schema.TestResourceDataRaw(t,
				map[string]*schema.Schema{POLICY_STATEMENTS: policyStatementsSchema(policyStatementSchemaOptions{})},
				tc.policyStatements,
			)

			schemaStatements, ok := statementsData.Get(POLICY_STATEMENTS).([]interface{})
			require.True(t, ok)
			actual, err := policyStatementsFromResourceData(schemaStatements)
			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)

			// with v7 of the go client there is an accidental duplicate type, so it returns a Statement type
			// even though it takes a StatementPost type
			actualRaw := policyStatementsToResourceData(statementsToStatementReps(statementPostsToStatements(actual)))
			require.Equal(t, tc.policyStatements[POLICY_STATEMENTS], actualRaw)
		})
	}
}

func TestPolicyStatementValidation(t *testing.T) {
	testCases := []struct {
		name              string
		statement         map[string]interface{}
		expectedErrString string
	}{
		{
			name: "both actions and not_actions",
			statement: map[string]interface{}{
				ACTIONS:       []interface{}{"*"},
				NOT_ACTIONS:   []interface{}{"updateOn"},
				RESOURCES:     []interface{}{"proj/*"},
				NOT_RESOURCES: []interface{}{},
				EFFECT:        "allow",
			},
			expectedErrString: "policy statements cannot contain both 'actions' and 'not_actions'",
		},
		{
			name: "both resources and not_resources",
			statement: map[string]interface{}{
				ACTIONS:       []interface{}{"*"},
				NOT_ACTIONS:   []interface{}{},
				NOT_RESOURCES: []interface{}{"webhook/*"},
				RESOURCES:     []interface{}{"proj/*"},
				EFFECT:        "allow",
			},
			expectedErrString: "policy statements cannot contain both 'resources' and 'not_resources'",
		},
		{
			name: "no actions or not_actions",
			statement: map[string]interface{}{
				ACTIONS:       []interface{}{},
				NOT_ACTIONS:   []interface{}{},
				RESOURCES:     []interface{}{"proj/*"},
				NOT_RESOURCES: []interface{}{},
				EFFECT:        "allow",
			},
			expectedErrString: "policy statements must contain either 'actions' or 'not_actions'",
		},
		{
			name: "no resources or not_resources",
			statement: map[string]interface{}{
				ACTIONS:       []interface{}{"*"},
				NOT_ACTIONS:   []interface{}{},
				RESOURCES:     []interface{}{},
				NOT_RESOURCES: []interface{}{},
				EFFECT:        "allow",
			},
			expectedErrString: "policy statements must contain either 'resources' or 'not_resources'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.EqualError(t, validatePolicyStatement(tc.statement), tc.expectedErrString)
		})
	}
}

// statementPostToStatement is a helper function just for these tests
// since v7 of the go client passes and returns two differing types
func statementPostsToStatements(posts []ldapi.StatementPost) []ldapi.Statement {
	var statements []ldapi.Statement
	for _, p := range posts {
		p := p
		statement := ldapi.Statement(p)
		statements = append(statements, statement)
	}
	return statements
}

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

func TestSuppressEquivalentJsonDiffs_StatementVariants(t *testing.T) {
	a := `[{"effect":"allow","resources":["proj/*:env/staging"],"actions":["*"]}]`
	b := `[
        {
            "actions": ["*"],
            "resources": ["proj/*:env/staging"],
            "effect": "allow"
        }
    ]`
	assert.True(t, suppressEquivalentJsonDiffs(POLICY_STATEMENTS_JSON, a, b, nil))

	c := `[{"effect":"deny","resources":["proj/*:env/staging"],"actions":["*"]}]`
	assert.False(t, suppressEquivalentJsonDiffs(POLICY_STATEMENTS_JSON, a, c, nil))

	assert.True(t, suppressEquivalentJsonDiffs(POLICY_STATEMENTS_JSON, "", "", nil))
}

func TestPolicyStatementsFromJSON_TolerantOfTrimmableWhitespace(t *testing.T) {
	input := "\n  [{\"effect\":\"allow\",\"resources\":[\"proj/*\"],\"actions\":[\"*\"]}]\n  "
	statements, err := policyStatementsFromJSON(input)
	require.NoError(t, err)
	require.Len(t, statements, 1)
	assert.True(t, strings.HasSuffix(input, "  "))
}
