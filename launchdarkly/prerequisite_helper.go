package launchdarkly

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go"
)

func prerequisitesSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				FLAG_KEY: {
					Type:         schema.TypeString,
					Required:     true,
					ValidateFunc: validateKey(),
				},
				VARIATION: &schema.Schema{
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
