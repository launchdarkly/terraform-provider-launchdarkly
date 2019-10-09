package launchdarkly

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"

	ldapi "github.com/launchdarkly/api-client-go"
)

func targetsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"values": {
					Type:     schema.TypeList,
					Elem:     &schema.Schema{Type: schema.TypeString},
					Optional: true,
				},
			},
		},
	}
}

func targetsFromResourceData(d *schema.ResourceData, metaRaw interface{}) []ldapi.Target {
	tgts, ok := d.GetOk(user_targets)
	if !ok {
		return []ldapi.Target{}
	}
	schemaTargets := tgts.([]interface{})
	targets := make([]ldapi.Target, len(schemaTargets))
	for i, target := range schemaTargets {
		v := targetFromResourceData(i, target)
		targets[i] = v
	}
	return targets
}

func targetFromResourceData(variation int, val interface{}) ldapi.Target {
	if val == nil {
		return ldapi.Target{Variation: int32(variation)}
	}
	targetMap := val.(map[string]interface{})
	p := ldapi.Target{
		Variation: int32(variation),
	}
	for _, v := range targetMap[values].([]interface{}) {
		p.Values = append(p.Values, v.(string))
	}

	log.Printf("[DEBUG] %+v\n", p)

	return p
}

func targetsToResourceData(targets []ldapi.Target) interface{} {
	transformed := make([]interface{}, len(targets))

	for _, p := range targets {
		transformed[p.Variation] = map[string]interface{}{
			values: p.Values,
		}
	}

	return transformed
}
