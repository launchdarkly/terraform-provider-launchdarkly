package launchdarkly

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"

	ldapi "github.com/launchdarkly/api-client-go"
)

func targetsSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		Description: "List of nested blocks describing the individual user targets for each variation. The order of the user_targets blocks determines the index of the variation to serve if a user_target is matched",
		Computed:    true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"values": {
					Type:        schema.TypeList,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Optional:    true,
					Description: "List of user strings to target",
				},
			},
		},
	}
}

func targetsFromResourceData(d *schema.ResourceData) []ldapi.Target {
	var schemaTargets []interface{}
	targetsHasChange := d.HasChange(TARGETS)
	userTargetsHasChange := d.HasChange(USER_TARGETS)
	if targetsHasChange {
		schemaTargets = d.Get(TARGETS).([]interface{})
	} else if userTargetsHasChange {
		schemaTargets = d.Get(USER_TARGETS).([]interface{})
	}
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

// targetToResourceData converts the `target` information returned
// by the LaunchDarkly API into a format suitable for Terraform
// If no `targets` are specified for a given variation, LaunchDarkly may
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
