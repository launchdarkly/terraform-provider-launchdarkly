package launchdarkly

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/launchdarkly/api-client-go"
	"github.com/pkg/errors"
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
				value: {
					Type:     schema.TypeString,
					Required: true,
				},
			},
		},
	}
}

func variationTypeSchema() *schema.Schema {
	return &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Default:      "bool",
		Description:  "Default: bool. Must be one of: float, bool, string, json",
		ValidateFunc: nil,
	}
}

func variationsFromResourceData(d *schema.ResourceData, variationType string) ([]ldapi.Variation, error) {
	schemaVariations := d.Get(variations).(*schema.Set)
	if variationType == "bool" {
		if schemaVariations.Len() > 0 {
			return nil, fmt.Errorf("%s=bool cannot be set along with variations! Please specify variation_type", variation_type)
		}
		return nil, nil
	}

	variations := make([]ldapi.Variation, schemaVariations.Len())
	list := schemaVariations.List()
	for i, variation := range list {
		v, err := variationFromResourceData(variation, variationType)
		if err != nil {
			return nil, err
		}
		variations[i] = v
	}
	return variations, nil
}

func variationFromResourceData(variation interface{}, variationType string) (ldapi.Variation, error) {
	variationMap := variation.(map[string]interface{})
	var val interface{}

	v := variationMap[value]
	// all types get represented as a string in terraform world.
	valueString := v.(string)
	var err error

	switch variationType {
	case "string":
		val = valueString
	case "float":
		val, err = strconv.ParseFloat(valueString, 64)
		if err != nil {
			return ldapi.Variation{}, errors.Wrapf(err, "Expected string: %q to parse as a float.", valueString)
		}
	case "json":
		var jsonMap map[string]interface{}
		err = json.Unmarshal([]byte(valueString), &jsonMap)
		if err != nil {
			return ldapi.Variation{}, errors.Wrapf(err, "Expected string: %q to be valid json.", valueString)
		}
		val = jsonMap
	default:
		return ldapi.Variation{}, fmt.Errorf("unexpected variation type: %q must be one of: string,float,json", variationType)
	}

	return ldapi.Variation{
		Name:        variationMap[name].(string),
		Description: variationMap[description].(string),
		Value:       &val,
	}, nil
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
func variationHash(val interface{}) int {
	variationMap := val.(map[string]interface{})
	return hashcode.String(fmt.Sprintf("%v", variationMap[value]))
}
