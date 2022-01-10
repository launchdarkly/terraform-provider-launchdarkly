package launchdarkly

import (
	"errors"

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
}

func policyStatementsSchema(options policyStatementSchemaOptions) *schema.Schema {
	return &schema.Schema{
		Type:          schema.TypeList,
		Optional:      true,
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
					Type:         schema.TypeString,
					Required:     true,
					ValidateFunc: validation.StringInSlice([]string{"allow", "deny"}, false),
				},
			},
		},
	}
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
		s := policyStatementFromResourceData(statement)
		statements = append(statements, s)
	}
	return statements, nil
}

func policyStatementFromResourceData(statement map[string]interface{}) ldapi.StatementPost {
	ret := ldapi.StatementPost{
		Effect: statement[EFFECT].(string),
	}
	for _, r := range statement[RESOURCES].([]interface{}) {
		ret.Resources = append(ret.Resources, r.(string))
	}
	for _, a := range statement[ACTIONS].([]interface{}) {
		ret.Actions = append(ret.Actions, a.(string))
	}
	// optional fields
	rawNotResources := statement[NOT_RESOURCES].([]interface{})
	var notResources []string
	for _, n := range rawNotResources {
		notResources = append(notResources, n.(string))
		ret.NotResources = &notResources
	}
	rawNotActions := statement[NOT_ACTIONS].([]interface{})
	var notActions []string
	for _, n := range rawNotActions {
		notActions = append(notActions, n.(string))
		ret.NotActions = &notActions
	}
	return ret
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
