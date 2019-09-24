package launchdarkly

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go"
)

func prerequisitesSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				flag_key: {
					Type:         schema.TypeString,
					Required:     true,
					ValidateFunc: validateKey(),
				},
				variation: &schema.Schema{
					Type:         schema.TypeInt,
					Elem:         &schema.Schema{Type: schema.TypeInt},
					Required:     true,
					ValidateFunc: validation.IntAtLeast(0),
				},
			},
		},
	}
}

func prerequisitesFromResourceData(d *schema.ResourceData, metaRaw interface{}) []ldapi.Prerequisite {
	schemaPrerequisites := d.Get(prerequisites).([]interface{})
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
		Key:       prerequisiteMap[flag_key].(string),
		Variation: int32(prerequisiteMap[variation].(int)),
	}

	log.Printf("[DEBUG] %+v\n", p)
	return p
}

func prerequisitesToResourceData(prerequisites []ldapi.Prerequisite) interface{} {
	transformed := make([]interface{}, len(prerequisites))

	for i, p := range prerequisites {
		transformed[i] = map[string]interface{}{
			flag_key:  p.Key,
			variation: p.Variation,
		}
	}
	return transformed
}
