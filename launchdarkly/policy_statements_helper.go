package launchdarkly

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	ldapi "github.com/launchdarkly/api-client-go/v22"
)

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
// policy statements into []ldapi.StatementPost. Expected shape mirrors the
// block schema (snake_case keys: resources, not_resources, actions,
// not_actions, effect). Returns (nil, nil) for empty input.
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
		effectRaw, ok := stmt[EFFECT]
		if !ok {
			return nil, fmt.Errorf("%s[%d]: 'effect' is required", POLICY_STATEMENTS_JSON, i)
		}
		effect, ok := effectRaw.(string)
		if !ok {
			return nil, fmt.Errorf("%s[%d]: 'effect' must be a string", POLICY_STATEMENTS_JSON, i)
		}
		resources := jsonStringSliceField(stmt, RESOURCES)
		notResources := jsonStringSliceField(stmt, NOT_RESOURCES)
		actions := jsonStringSliceField(stmt, ACTIONS)
		notActions := jsonStringSliceField(stmt, NOT_ACTIONS)
		s, err := jsonPolicyStatementToLDAPI(effect, resources, notResources, actions, notActions)
		if err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", POLICY_STATEMENTS_JSON, i, err)
		}
		statements = append(statements, s)
	}
	return statements, nil
}

func jsonPolicyStatementToLDAPI(effect string, resources, notResources, actions, notActions []string) (ldapi.StatementPost, error) {
	if len(resources) > 0 && len(notResources) > 0 {
		return ldapi.StatementPost{}, errors.New("policy statements cannot contain both 'resources' and 'not_resources'")
	}
	if len(resources) == 0 && len(notResources) == 0 {
		return ldapi.StatementPost{}, errors.New("policy statements must contain either 'resources' or 'not_resources'")
	}
	if len(actions) > 0 && len(notActions) > 0 {
		return ldapi.StatementPost{}, errors.New("policy statements cannot contain both 'actions' and 'not_actions'")
	}
	if len(actions) == 0 && len(notActions) == 0 {
		return ldapi.StatementPost{}, errors.New("policy statements must contain either 'actions' or 'not_actions'")
	}
	stmt := ldapi.StatementPost{Effect: effect}
	if len(resources) > 0 {
		stmt.SetResources(resources)
	}
	if len(notResources) > 0 {
		stmt.SetNotResources(notResources)
	}
	if len(actions) > 0 {
		stmt.SetActions(actions)
	}
	if len(notActions) > 0 {
		stmt.SetNotActions(notActions)
	}
	return stmt, nil
}

// jsonStringSliceField coerces a JSON array of strings (decoded into
// []interface{}) into []string. Returns an empty slice for missing,
// null, or non-array values. Non-string elements are skipped.
func jsonStringSliceField(stmt map[string]interface{}, key string) []string {
	raw, ok := stmt[key]
	if !ok || raw == nil {
		return nil
	}
	arr, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
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
