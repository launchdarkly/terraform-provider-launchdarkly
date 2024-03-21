package launchdarkly

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go/v15"
)

func prerequisitesSchema(isDataSource bool) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    !isDataSource,
		Computed:    isDataSource,
		Description: "List of nested blocks describing prerequisite feature flags rules.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				FLAG_KEY: {
					Type:             schema.TypeString,
					Required:         true,
					Description:      "The prerequisite feature flag's `key`.",
					ValidateDiagFunc: validateKey(),
				},
				VARIATION: {
					Type:             schema.TypeInt,
					Elem:             &schema.Schema{Type: schema.TypeInt},
					Required:         true,
					Description:      "The index of the prerequisite feature flag's variation to target.",
					ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(0)),
				},
			},
		},
	}
}

func prerequisitesFromResourceData(d *schema.ResourceData, metaRaw interface{}) []ldapi.Prerequisite {
	schemaPrerequisites := d.Get(PREREQUISITES).([]interface{})
	prerequisites := make([]ldapi.Prerequisite, len(schemaPrerequisites))
	for i, prerequisite := range schemaPrerequisites {
		v := prerequisiteFromResourceData(prerequisite)
		prerequisites[i] = v
	}

	return prerequisites
}

func prerequisiteFromResourceData(val interface{}) ldapi.Prerequisite {
	prerequisiteMap := val.(map[string]interface{})
	p := ldapi.Prerequisite{
		Key:       prerequisiteMap[FLAG_KEY].(string),
		Variation: int32(prerequisiteMap[VARIATION].(int)),
	}

	log.Printf("[DEBUG] %+v\n", p)
	return p
}

func prerequisitesToResourceData(prerequisites []ldapi.Prerequisite) interface{} {
	transformed := make([]interface{}, 0, len(prerequisites))
	for _, prereq := range prerequisites {
		transformed = append(transformed, map[string]interface{}{
			FLAG_KEY:  prereq.Key,
			VARIATION: prereq.Variation,
		})
	}
	return transformed
}
