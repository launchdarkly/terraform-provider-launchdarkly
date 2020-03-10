package launchdarkly

import (
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
)

func policyArraySchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Set:      policyHash,
		Optional: true,
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

func policiesFromResourceData(d *schema.ResourceData) []ldapi.Policy {
	schemaPolicies := d.Get(POLICY).(*schema.Set)

	policies := make([]ldapi.Policy, schemaPolicies.Len())
	list := schemaPolicies.List()
	for i, policy := range list {
		v := policyFromResourceData(policy)
		policies[i] = v
	}
	return policies
}

func policyFromResourceData(val interface{}) ldapi.Policy {
	policyMap := val.(map[string]interface{})
	p := ldapi.Policy{
		Resources: []string{},
		Actions:   []string{},
		Effect:    policyMap[EFFECT].(string),
	}
	for _, r := range policyMap[RESOURCES].([]interface{}) {
		p.Resources = append(p.Resources, r.(string))
	}
	for _, a := range policyMap[ACTIONS].([]interface{}) {
		p.Actions = append(p.Actions, a.(string))
	}

	sort.Strings(p.Actions)
	sort.Strings(p.Resources)
	return p
}

func policiesToResourceData(policies []ldapi.Policy) interface{} {
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
func policyHash(val interface{}) int {
	policy := policyFromResourceData(val)
	return hashcode.String(fmt.Sprintf("%v", policy))
}
