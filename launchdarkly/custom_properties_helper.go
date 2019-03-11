package launchdarkly

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/launchdarkly/api-client-go"
	"sort"
)

func customPropertiesSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Set:      customPropertyHash,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				key: {
					Type:     schema.TypeString,
					Required: true,
				},
				name: {
					Type:     schema.TypeString,
					Required: true,
				},
				value: {
					Type: schema.TypeList,
					//Set:      stringHash,
					Required: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},
			},
		},
	}
}

func customPropertiesFromResourceData(d *schema.ResourceData) map[string]ldapi.CustomProperty {
	customPropertiesRaw := d.Get(custom_properties)
	schemaCustomProperties := customPropertiesRaw.(*schema.Set)
	customProperties := make(map[string]ldapi.CustomProperty)
	for _, cpRaw := range schemaCustomProperties.List() {
		key, cp := customPropertyFromResourceData(cpRaw)
		customProperties[key] = cp
	}
	return customProperties
}

func customPropertyFromResourceData(val interface{}) (string, ldapi.CustomProperty) {
	customPropertyMap := val.(map[string]interface{})

	var values []string
	for _, v := range customPropertyMap[value].([]interface{}) {
		values = append(values, v.(string))
	}
	sort.Strings(values)

	cp := ldapi.CustomProperty{
		Name:  customPropertyMap[name].(string),
		Value: values,
	}

	return customPropertyMap[key].(string), cp
}

func customPropertiesToResourceData(customProperties map[string]ldapi.CustomProperty) []interface{} {
	transformed := make([]interface{}, 0)

	for k, cp := range customProperties {
		cpRaw := map[string]interface{}{
			key:   k,
			name:  cp.Name,
			value: cp.Value,
		}
		transformed = append(transformed, cpRaw)
	}
	return transformed
}

// https://godoc.org/github.com/hashicorp/terraform/helper/schema#SchemaSetFunc
func customPropertyHash(val interface{}) int {
	customPropertyMap := val.(map[string]interface{})
	return hashcode.String(fmt.Sprintf("%v", customPropertyMap[key]))
}
