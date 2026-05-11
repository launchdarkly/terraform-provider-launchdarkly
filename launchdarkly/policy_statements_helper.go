package launchdarkly

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

// policyStatementSchemaOptions is used to help with renaming 'policy_statements' to statements for the launchdarkly_webhook resource.
// This struct can be removed after we have dropped support for 'policy_statements'
type policyStatementSchemaOptions struct {
	// when set, the attribute will be marked as 'deprecated'
	deprecated    string
	description   string
	conflictsWith []string
	required      bool
	optional      bool
	computed      bool
}

func policyStatementsSchema(options policyStatementSchemaOptions) *schema.Schema {
	minItems := 0
	if !options.computed {
		minItems = 1
	}
	schema := &schema.Schema{
		Type:          schema.TypeList,
		Optional:      options.optional,
		Required:      options.required,
		Computed:      options.computed,
		MinItems:      minItems,
		Description:   options.description,
		Deprecated:    options.deprecated,
		ConflictsWith: options.conflictsWith,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				RESOURCES: {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Optional:    true,
					Description: "The list of resource specifiers defining the resources to which the statement applies.",
					MinItems:    1,
				},
				NOT_RESOURCES: {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Optional:    true,
					Description: "The list of resource specifiers defining the resources to which the statement does not apply.",
					MinItems:    1,
				},
				ACTIONS: {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Optional:    true,
					Description: "The list of action specifiers defining the actions to which the statement applies.\nEither `actions` or `not_actions` must be specified. For a list of available actions read [Actions reference](https://docs.launchdarkly.com/home/account-security/custom-roles/actions#actions-reference).",
					MinItems:    1,
				},
				NOT_ACTIONS: {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Optional:    true,
					Description: "The list of action specifiers defining the actions to which the statement does not apply.",
					MinItems:    1,
				},
				EFFECT: {
					Type:             schema.TypeString,
					Required:         true,
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"allow", "deny"}, false)),
					Description:      "Either `allow` or `deny`. This argument defines whether the statement allows or denies access to the named resources and actions.",
				},
			},
		},
	}
	return schema
}

func validatePolicyStatement(statement map[string]interface{}) error {
	resources := statement[RESOURCES].([]interface{})
	notResources := statement[NOT_RESOURCES].([]interface{})
	actions := statement[ACTIONS].([]interface{})
	notActions := statement[NOT_ACTIONS].([]interface{})
	if len(resources) > 0 && len(notResources) > 0 {
		return errors.New("policy statements cannot contain both 'resources' and 'not_resources'")
	}
	if len(resources) == 0 && len(notResources) == 0 {
		return errors.New("policy statements must contain either 'resources' or 'not_resources'")
	}
	if len(actions) > 0 && len(notActions) > 0 {
		return errors.New("policy statements cannot contain both 'actions' and 'not_actions'")
	}
	if len(actions) == 0 && len(notActions) == 0 {
		return errors.New("policy statements must contain either 'actions' or 'not_actions'")
	}
	return nil
}

func policyStatementsFromResourceData(schemaStatements []interface{}) ([]ldapi.StatementPost, error) {
	statements := make([]ldapi.StatementPost, 0, len(schemaStatements))
	for _, stmt := range schemaStatements {
		statement := stmt.(map[string]interface{})
		s, err := policyStatementFromResourceData(statement)
		if err != nil {
			return statements, err
		}
		statements = append(statements, s)
	}
	return statements, nil
}

func policyStatementFromResourceData(statement map[string]interface{}) (ldapi.StatementPost, error) {
	err := validatePolicyStatement(statement)
	if err != nil {
		return ldapi.StatementPost{}, err
	}
	ret := ldapi.StatementPost{
		Effect: statement[EFFECT].(string),
	}
	resources := interfaceSliceToStringSlice(statement[RESOURCES].([]interface{}))
	if len(resources) > 0 {
		ret.SetResources(resources)
	}
	notResources := interfaceSliceToStringSlice(statement[NOT_RESOURCES].([]interface{}))
	if len(notResources) > 0 {
		ret.SetNotResources(notResources)
	}
	actions := interfaceSliceToStringSlice(statement[ACTIONS].([]interface{}))
	if len(actions) > 0 {
		ret.SetActions(actions)
	}
	notActions := interfaceSliceToStringSlice(statement[NOT_ACTIONS].([]interface{}))
	if len(notActions) > 0 {
		ret.SetNotActions(notActions)
	}

	return ret, nil
}

func policyStatementsToResourceData(statements []ldapi.Statement) []interface{} {
	transformed := make([]interface{}, 0, len(statements))
	for _, s := range statements {
		t := map[string]interface{}{
			EFFECT: s.Effect,
		}
		if len(s.Resources) > 0 {
			t[RESOURCES] = stringSliceToInterfaceSlice(s.Resources)
		}
		if len(s.NotResources) > 0 {
			t[NOT_RESOURCES] = stringSliceToInterfaceSlice(s.NotResources)
		}
		if len(s.Actions) > 0 {
			t[ACTIONS] = stringSliceToInterfaceSlice(s.Actions)
		}
		if len(s.NotActions) > 0 {
			t[NOT_ACTIONS] = stringSliceToInterfaceSlice(s.NotActions)
		}
		transformed = append(transformed, t)
	}
	return transformed
}

func statementsToStatementReps(policies []ldapi.Statement) []ldapi.Statement {
	statements := make([]ldapi.Statement, 0, len(policies))
	for _, p := range policies {
		rep := ldapi.Statement(p)
		statements = append(statements, rep)
	}
	return statements
}

// The relay proxy config api requires a statementRep in the POST body
func statementPostsToStatementReps(policies []ldapi.StatementPost) []ldapi.Statement {
	statements := make([]ldapi.Statement, 0, len(policies))
	for _, p := range policies {
		rep := ldapi.Statement(p)
		statements = append(statements, rep)
	}
	return statements
}

// policyStatementsFromJSON decodes a JSON document representing an array of
// policy statements into the same []ldapi.StatementPost slice the block form
// produces. Expected shape mirrors the block schema (snake_case keys:
// resources, not_resources, actions, not_actions, effect). Returns (nil, nil)
// for empty input.
func policyStatementsFromJSON(raw string) ([]ldapi.StatementPost, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var decoded []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, fmt.Errorf("%s: invalid JSON: %w", POLICY_STATEMENTS_JSON, err)
	}
	statements := make([]ldapi.StatementPost, 0, len(decoded))
	for i, stmt := range decoded {
		normalized := map[string]interface{}{
			RESOURCES:     jsonStringSliceField(stmt, RESOURCES),
			NOT_RESOURCES: jsonStringSliceField(stmt, NOT_RESOURCES),
			ACTIONS:       jsonStringSliceField(stmt, ACTIONS),
			NOT_ACTIONS:   jsonStringSliceField(stmt, NOT_ACTIONS),
		}
		effectRaw, ok := stmt[EFFECT]
		if !ok {
			return nil, fmt.Errorf("%s[%d]: 'effect' is required", POLICY_STATEMENTS_JSON, i)
		}
		effect, ok := effectRaw.(string)
		if !ok {
			return nil, fmt.Errorf("%s[%d]: 'effect' must be a string", POLICY_STATEMENTS_JSON, i)
		}
		normalized[EFFECT] = effect
		s, err := policyStatementFromResourceData(normalized)
		if err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", POLICY_STATEMENTS_JSON, i, err)
		}
		statements = append(statements, s)
	}
	return statements, nil
}

// jsonStringSliceField coerces a JSON array of strings, decoded by
// encoding/json into []interface{}, into the []interface{} shape that
// policyStatementFromResourceData expects. Returns an empty slice for missing
// or non-array keys.
func jsonStringSliceField(stmt map[string]interface{}, key string) []interface{} {
	raw, ok := stmt[key]
	if !ok || raw == nil {
		return []interface{}{}
	}
	arr, ok := raw.([]interface{})
	if !ok {
		return []interface{}{}
	}
	return arr
}

// policyStatementsToJSON converts the LD API response back into the canonical
// JSON form for storage in policy_statements_json. Returns "" when statements
// is empty so an unset attribute stays unset.
func policyStatementsToJSON(statements []ldapi.Statement) (string, error) {
	if len(statements) == 0 {
		return "", nil
	}
	out := make([]map[string]interface{}, 0, len(statements))
	for _, s := range statements {
		m := map[string]interface{}{
			EFFECT: s.Effect,
		}
		if len(s.Resources) > 0 {
			m[RESOURCES] = s.Resources
		}
		if len(s.NotResources) > 0 {
			m[NOT_RESOURCES] = s.NotResources
		}
		if len(s.Actions) > 0 {
			m[ACTIONS] = s.Actions
		}
		if len(s.NotActions) > 0 {
			m[NOT_ACTIONS] = s.NotActions
		}
		out = append(out, m)
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", fmt.Errorf("%s: failed to marshal: %w", POLICY_STATEMENTS_JSON, err)
	}
	return string(b), nil
}
