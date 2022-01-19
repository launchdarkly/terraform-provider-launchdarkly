package launchdarkly

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go/v7"
)

// policyStatementSchemaOptions is used to help with renaming 'policy_statements' to statements for the launchdarkly_webhook resource.
// This struct can be removed after we have dropped support for 'policy_statements'
type policyStatementSchemaOptions struct {
	// when set, the attribute will be marked as 'deprected'
	deprecated    string
	description   string
	conflictsWith []string
	required      bool
}

func policyStatementsSchema(options policyStatementSchemaOptions) *schema.Schema {
	schema := &schema.Schema{
		Type:          schema.TypeList,
		Optional:      !options.required,
		Required:      options.required,
		MinItems:      1,
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
					Description: "A list of LaunchDarkly resource specifiers",
					MinItems:    1,
				},
				NOT_RESOURCES: {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Optional:    true,
					Description: "Targeted resources will be those resources NOT in this list. The 'resources' field must be empty to use this field",
					MinItems:    1,
				},
				ACTIONS: {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Optional:    true,
					Description: "An action to perform on a resource",
					MinItems:    1,
				},
				NOT_ACTIONS: {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Optional:    true,
					Description: "Targeted actions will be those actions NOT in this list. The 'actions' field must be empty to use this field",
					MinItems:    1,
				},
				EFFECT: {
					Type:             schema.TypeString,
					Required:         true,
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"allow", "deny"}, false)),
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
		err := validatePolicyStatement(statement)
		if err != nil {
			return statements, err
		}
		s, err := policyStatementFromResourceData(statement)
		if err != nil {
			return statements, err
		}
		statements = append(statements, s)
	}
	return statements, nil
}

func policyStatementFromResourceData(statement map[string]interface{}) (ldapi.StatementPost, error) {
	statementResources := []string{}
	statementActions := []string{}
	statementNotResources := []string{}
	statementNotActions := []string{}
	ret := ldapi.StatementPost{
		Effect: statement[EFFECT].(string),
	}
	// Build our policy fields
	for _, r := range statement[RESOURCES].([]interface{}) {
		statementResources = append(statementResources, r.(string))
	}
	for _, a := range statement[ACTIONS].([]interface{}) {
		statementActions = append(statementActions, a.(string))
	}
	// optional fields
	rawNotResources := statement[NOT_RESOURCES].([]interface{})
	for _, n := range rawNotResources {
		statementNotResources = append(statementNotResources, n.(string))
	}
	rawNotActions := statement[NOT_ACTIONS].([]interface{})
	for _, n := range rawNotActions {
		statementNotActions = append(statementNotActions, n.(string))
	}
	// Add the appropriate fields to the statement
	if len(statement[RESOURCES].([]interface{})) > 0 {
		ret.Resources = &statementResources
	} else if len(statement[NOT_RESOURCES].([]interface{})) > 0 {
		ret.NotResources = &statementNotResources
	} else {
		return ret, fmt.Errorf("please provide either 'resources' or not_resources' for your policy_statement")
	}
	if len(statement[ACTIONS].([]interface{})) > 0 {
		ret.Actions = &statementActions
	} else if len(statement[NOT_ACTIONS].([]interface{})) > 0 {
		ret.NotActions = &statementNotActions
	} else {
		return ret, fmt.Errorf("please provide either 'actions' or not_actions' for your policy_statement")
	}

	return ret, nil
}

func policyStatementsToResourceData(statements []ldapi.StatementRep) []interface{} {
	transformed := make([]interface{}, 0, len(statements))
	for _, s := range statements {
		t := map[string]interface{}{
			EFFECT: s.Effect,
		}
		if s.Resources != nil && len(*s.Resources) > 0 {
			var resources []interface{}
			for _, v := range *s.Resources {
				resources = append(resources, v)
			}
			t[RESOURCES] = resources
		}
		if s.NotResources != nil && len(*s.NotResources) > 0 {
			var notResources []interface{}
			for _, v := range *s.NotResources {
				notResources = append(notResources, v)
			}
			t[NOT_RESOURCES] = notResources
		}
		if s.Actions != nil && len(*s.Actions) > 0 {
			t[ACTIONS] = stringSliceToInterfaceSlice(*s.Actions)
		}
		if s.NotActions != nil && len(*s.NotActions) > 0 {
			t[NOT_ACTIONS] = stringSliceToInterfaceSlice(*s.NotActions)
		}
		transformed = append(transformed, t)
	}
	return transformed
}

func statementsToStatementReps(policies []ldapi.Statement) []ldapi.StatementRep {
	statements := make([]ldapi.StatementRep, 0, len(policies))
	for _, p := range policies {
		rep := ldapi.StatementRep(p)
		statements = append(statements, rep)
	}
	return statements
}

// The relay proxy config api requires a statementRep in the POST body
func statementPostsToStatementReps(policies []ldapi.StatementPost) []ldapi.StatementRep {
	statements := make([]ldapi.StatementRep, 0, len(policies))
	for _, p := range policies {
		rep := ldapi.StatementRep(p)
		statements = append(statements, rep)
	}
	return statements
}
