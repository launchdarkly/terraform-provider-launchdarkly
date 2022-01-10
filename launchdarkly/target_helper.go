package launchdarkly

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	ldapi "github.com/launchdarkly/api-client-go/v7"
)

func targetsSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeSet,
		Optional:    true,
		Description: "Set of nested blocks describing the individual user targets for each variation",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				VALUES: {
					Type:        schema.TypeList,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Required:    true,
					Description: "List of user strings to target",
				},
				VARIATION: {
					Type:             schema.TypeInt,
					Required:         true,
					Description:      "Index of the variation to serve if a user_target is matched",
					ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(0)),
				},
			},
		},
	}
}

func targetsFromResourceData(d *schema.ResourceData) []ldapi.Target {
	tgts, ok := d.GetOk(TARGETS)
	if !ok {
		return []ldapi.Target{}
	}
	schemaTargets := tgts.(*schema.Set).List()
	targets := make([]ldapi.Target, 0, len(schemaTargets))
	for _, target := range schemaTargets {
		targetMap := target.(map[string]interface{})
		targets = append(targets, targetFromResourceData(targetMap))
	}
	return targets
}

func targetFromResourceData(targetMap map[string]interface{}) ldapi.Target {
	resourceValues := targetMap[VALUES].([]interface{})
	values := make([]string, 0, len(resourceValues))
	for _, v := range resourceValues {
		values = append(values, v.(string))
	}
	return ldapi.Target{
		Variation: int32(targetMap[VARIATION].(int)),
		Values:    values,
	}
}

// targetToResourceData converts the `target` information returned
// by the LaunchDarkly API into a format suitable for Terraform
func targetsToResourceData(targets []ldapi.Target) []interface{} {
	transformed := make([]interface{}, 0, len(targets))
	for _, target := range targets {
		resourceTarget := map[string]interface{}{
			VALUES:    target.Values,
			VARIATION: int(target.Variation),
		}
		transformed = append(transformed, resourceTarget)
	}
	return transformed
}
