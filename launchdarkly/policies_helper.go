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
				resources: {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Required: true,
				},
				actions: {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Required: true,
				},
				effect: {
					Type:     schema.TypeString,
					Required: true,
				},
			},
		},
	}
}

func policiesFromResourceData(d *schema.ResourceData) []ldapi.Policy {
	schemaPolicies := d.Get(policy).(*schema.Set)

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
		Effect:    policyMap[effect].(string),
	}
	for _, r := range policyMap[resources].([]interface{}) {
		p.Resources = append(p.Resources, r.(string))
	}
	for _, a := range policyMap[actions].([]interface{}) {
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
			resources: p.Resources,
			actions:   p.Actions,
			effect:    p.Effect,
		}
	}
	return transformed
}

// https://godoc.org/github.com/hashicorp/terraform/helper/schema#SchemaSetFunc
func policyHash(val interface{}) int {
	policy := policyFromResourceData(val)
	return hashcode.String(fmt.Sprintf("%v", policy))
}
