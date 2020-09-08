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
		Computed: true,
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
	tgts, ok := d.GetOk(USER_TARGETS)
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
	for _, v := range targetMap[VALUES].([]interface{}) {
		p.Values = append(p.Values, v.(string))
	}

	log.Printf("[DEBUG] %+v\n", p)

	return p
}

// targetToResourceData converts the user_target information returned
// by the LaunchDarkly API into a format suitable for Terraform
// If no user_targets are specified for a given variation, LaunchDarkly may
// omit this information in the response. For example:
// "targets": [
// 	{
// 		"values": [
// 	  		"test"
// 		],
// 		"variation": 1
//   }
// ],
// From this information, we must imply that variation 0 has no targets.
func targetsToResourceData(targets []ldapi.Target) []interface{} {
	targetMap := make(map[int32][]string, len(targets))
	maxVariationIndex := int32(-1)

	for _, p := range targets {
		if p.Variation > maxVariationIndex {
			maxVariationIndex = p.Variation
		}
		targetMap[p.Variation] = p.Values
	}
	transformed := make([]interface{}, maxVariationIndex+1)

	for i := int32(0); i <= maxVariationIndex; i++ {
		values, found := targetMap[i]
		if !found {
			values = []string{}
		}
		transformed[i] = map[string]interface{}{
			VALUES: values,
		}
	}

	return transformed
}
