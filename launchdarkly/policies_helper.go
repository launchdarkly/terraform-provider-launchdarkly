package launchdarkly

import (
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v12"
)

func policyArraySchema() *schema.Schema {
	return &schema.Schema{
		Type:       schema.TypeSet,
		Set:        policyHash,
		Optional:   true,
		Deprecated: "'policy' is now deprecated. Please migrate to 'policy_statements' to maintain future compatability.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				RESOURCES: {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Required: true,
				},
				ACTIONS: {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Required: true,
				},
				EFFECT: {
					Type:     schema.TypeString,
					Required: true,
				},
			},
		},
	}
}

func policiesFromResourceData(d *schema.ResourceData) []ldapi.StatementPost {
	schemaPolicies := d.Get(POLICY).(*schema.Set)

	policies := make([]ldapi.StatementPost, schemaPolicies.Len())
	list := schemaPolicies.List()
	for i, policy := range list {
		v := policyFromResourceData(policy)
		policies[i] = v
	}
	return policies
}

func policyFromResourceData(val interface{}) ldapi.StatementPost {
	policyMap := val.(map[string]interface{})
	statementResources := []string{}
	statementActions := []string{}

	for _, r := range policyMap[RESOURCES].([]interface{}) {
		statementResources = append(statementResources, r.(string))
	}
	for _, a := range policyMap[ACTIONS].([]interface{}) {
		statementActions = append(statementActions, a.(string))
	}

	sort.Strings(statementActions)
	sort.Strings(statementResources)

	p := ldapi.StatementPost{
		Resources: statementResources,
		Actions:   statementActions,
		Effect:    policyMap[EFFECT].(string),
	}
	return p
}

func policiesToResourceData(policies []ldapi.Statement) interface{} {
	transformed := make([]interface{}, len(policies))

	for i, p := range policies {
		transformed[i] = map[string]interface{}{
			RESOURCES: p.Resources,
			ACTIONS:   p.Actions,
			EFFECT:    p.Effect,
		}
	}
	return transformed
}

// https://godoc.org/github.com/hashicorp/terraform/helper/schema#SchemaSetFunc
type hashStatement struct {
	Resources []string
	Actions   []string
	Effect    string
}

// https://godoc.org/github.com/hashicorp/terraform/helper/schema#SchemaSetFunc
func policyHash(val interface{}) int {
	rawPolicy := policyFromResourceData(val)
	// since this function runs once for each sub-field (unclear why)
	// it was creating 3 different hash indices per policy since it was hashing the
	// pointer addresses rather than the values themselves
	policy := hashStatement{
		Resources: rawPolicy.Resources,
		Actions:   rawPolicy.Actions,
		Effect:    rawPolicy.Effect,
	}
	return schema.HashString(fmt.Sprintf("%v", policy))
}
