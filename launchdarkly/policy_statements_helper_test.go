package launchdarkly

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicyStatementsRoundTripConversion(t *testing.T) {
	testCases := []struct {
		name             string
		policyStatements map[string]interface{}
		expected         []ldapi.Statement
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
			expected: []ldapi.Statement{
				{
					Resources: []string{"proj/*"},
					Actions:   []string{"*"},
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
			expected: []ldapi.Statement{
				{
					Resources: []string{"proj/*:env/*;qa_*"},
					Actions:   []string{"*"},
					Effect:    "allow",
				},
				{
					Resources: []string{"proj/*:env/*;qa_*:/flag/*"},
					Actions:   []string{"*"},
					Effect:    "allow",
				},
			},
		},
		{
			name: "not_resource example",
			policyStatements: map[string]interface{}{
				POLICY_STATEMENTS: []interface{}{
					map[string]interface{}{
						NOT_RESOURCES: []interface{}{"proj/*:env/production:flag/*"},
						ACTIONS:       []interface{}{"*"},
						EFFECT:        "allow",
					},
				},
			},
			expected: []ldapi.Statement{
				{
					NotResources: []string{"proj/*:env/production:flag/*"},
					Actions:      []string{"*"},
					Effect:       "allow",
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

			actualRaw := policyStatementsToResourceData(actual)
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
