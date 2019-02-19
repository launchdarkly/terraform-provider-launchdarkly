package launchdarkly

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
)

func variationsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Set:      variationHash,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				name: {
					Type:     schema.TypeString,
					Optional: true,
				},
				description: {
					Type:     schema.TypeString,
					Optional: true,
				},
				//TODO: enable other types by specifying them explicitly in the schema and then enforcing the type for each variation
				value: {
					Type:     schema.TypeString,
					Required: true,
				},
			},
		},
	}
}

func variationsFromResourceData(d *schema.ResourceData) []ldapi.Variation {
	schemaVariations := d.Get(variations).(*schema.Set)

	variations := make([]ldapi.Variation, schemaVariations.Len())
	for i, variation := range schemaVariations.List() {
		variations[i] = variationFromResourceData(variation)
	}
	return variations
}

func variationFromResourceData(variation interface{}) ldapi.Variation {
	variationMap := variation.(map[string]interface{})
	return ldapi.Variation{
		Name:        variationMap[name].(string),
		Description: variationMap[description].(string),
		Value:       ptr(variationMap[value]),
	}
}

func variationsToResourceData(variations []ldapi.Variation) interface{} {
	transformed := make([]interface{}, len(variations))

	for i, variation := range variations {
		transformed[i] = map[string]interface{}{
			name:        variation.Name,
			description: variation.Description,
			value:       variation.Value,
		}
	}
	return transformed
}

// https://godoc.org/github.com/hashicorp/terraform/helper/schema#SchemaSetFunc
func variationHash(value interface{}) int {
	v := variationFromResourceData(value).Value
	if v == nil {
		return 0
	}
	return hashcode.String(fmt.Sprintf("%v", *v))
}
