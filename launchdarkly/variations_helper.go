package launchdarkly

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/launchdarkly/api-client-go"
)

func variationsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Set:      variationHash,
		Optional: true,
		Computed: true,
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
				value: {
					Type:     schema.TypeString,
					Required: true,
					StateFunc: func(i interface{}) string {
						// LD allows arbitrary types here (*interface{}), but terraform wants a strong type here
						// As a compromise we only really support bool (default) and strings which works fine using this
						// technique:
						return fmt.Sprintf("%v", i)
					},
				},
			},
		},
	}
}

func variationsFromResourceData(d *schema.ResourceData) []ldapi.Variation {
	schemaVariations := d.Get(variations).(*schema.Set)

	variations := make([]ldapi.Variation, schemaVariations.Len())
	list := schemaVariations.List()

	// special case of boolean variations:
	if len(list) == 2 {
		values := []string{
			list[0].(map[string]interface{})[value].(string),
			list[1].(map[string]interface{})[value].(string),
		}
		if values[0] == "true" && values[1] == "false" {
			variations[0] = ldapi.Variation{Value: ptr(true)}
			variations[1] = ldapi.Variation{Value: ptr(false)}
			return variations
		}
	}

	for i, variation := range list {
		variations[i] = variationFromResourceData(variation)
	}
	return variations
}

func variationFromResourceData(variation interface{}) ldapi.Variation {
	variationMap := variation.(map[string]interface{})
	v := variationMap[value]
	return ldapi.Variation{
		Name:        variationMap[name].(string),
		Description: variationMap[description].(string),
		Value:       &v,
	}
}

func variationsToResourceData(variations []ldapi.Variation) interface{} {
	transformed := make([]interface{}, len(variations))

	for i, variation := range variations {
		transformed[i] = map[string]interface{}{
			name:        variation.Name,
			description: variation.Description,
			value:       fmt.Sprintf("%v", *variation.Value),
		}
	}
	return transformed
}

// https://godoc.org/github.com/hashicorp/terraform/helper/schema#SchemaSetFunc
func variationHash(val interface{}) int {
	variationMap := val.(map[string]interface{})
	return hashcode.String(fmt.Sprintf("%v", variationMap[value]))
}
