package launchdarkly

import (
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go/v17"
)

// https://docs.launchdarkly.com/home/connecting/custom-properties
const CUSTOM_PROPERTY_CHAR_LIMIT = 64
const CUSTOM_PROPERTY_ITEM_LIMIT = 64

func customPropertiesSchema(isDataSource bool) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeSet,
		Optional:    !isDataSource,
		Computed:    isDataSource,
		Set:         customPropertyHash,
		MaxItems:    CUSTOM_PROPERTY_ITEM_LIMIT,
		Description: "List of nested blocks describing the feature flag's [custom properties](https://docs.launchdarkly.com/home/connecting/custom-properties)",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				KEY: {
					Description:      "The unique custom property key.",
					Type:             schema.TypeString,
					Required:         true,
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(1, CUSTOM_PROPERTY_CHAR_LIMIT)),
				},
				NAME: {
					Description:      "The name of the custom property.",
					Type:             schema.TypeString,
					Required:         true,
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(1, CUSTOM_PROPERTY_CHAR_LIMIT)),
				},
				VALUE: {
					Type:        schema.TypeList,
					Required:    true,
					MaxItems:    CUSTOM_PROPERTY_ITEM_LIMIT,
					Description: "The list of custom property value strings.",
					Elem: &schema.Schema{
						Type: schema.TypeString,
						// Can't use validation.ToDiagFunc converted validators on TypeList at the moment
						// https://github.com/hashicorp/terraform-plugin-sdk/issues/734
						ValidateFunc: validation.StringLenBetween(1, CUSTOM_PROPERTY_CHAR_LIMIT),
					},
				},
			},
		},
	}
}

func customPropertiesFromResourceData(d *schema.ResourceData) map[string]ldapi.CustomProperty {
	customPropertiesRaw := d.Get(CUSTOM_PROPERTIES)
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
	for _, v := range customPropertyMap[VALUE].([]interface{}) {
		values = append(values, v.(string))
	}
	sort.Strings(values)

	cp := ldapi.CustomProperty{
		Name:  customPropertyMap[NAME].(string),
		Value: values,
	}

	return customPropertyMap[KEY].(string), cp
}

func customPropertiesToResourceData(customProperties map[string]ldapi.CustomProperty) []interface{} {
	transformed := make([]interface{}, 0)

	for k, cp := range customProperties {
		var values []interface{}
		for _, v := range cp.Value {
			values = append(values, v)
		}
		// Sort the values to ensure consistency with how they're sent to the API
		sort.Slice(values, func(i, j int) bool {
			return values[i].(string) < values[j].(string)
		})
		cpRaw := map[string]interface{}{
			KEY:   k,
			NAME:  cp.Name,
			VALUE: values,
		}
		transformed = append(transformed, cpRaw)
	}
	return transformed
}

// hashCustomProperty is a struct used for hashing custom properties
// to ensure consistent hash values based on actual field values
type hashCustomProperty struct {
	Key   string
	Name  string
	Value []string
}

// https://godoc.org/github.com/hashicorp/terraform/helper/schema#SchemaSetFunc
func customPropertyHash(val interface{}) int {
	customPropertyMap := val.(map[string]interface{})
	// Extract and sort values to ensure consistent hashing
	var values []string
	if valueList, ok := customPropertyMap[VALUE].([]interface{}); ok {
		for _, v := range valueList {
			values = append(values, v.(string))
		}
	}
	sort.Strings(values)

	// Hash all fields together to ensure consistent hash calculation
	// This prevents issues where the hash function is called with incomplete data
	cp := hashCustomProperty{
		Key:   fmt.Sprintf("%v", customPropertyMap[KEY]),
		Name:  fmt.Sprintf("%v", customPropertyMap[NAME]),
		Value: values,
	}
	return schema.HashString(fmt.Sprintf("%v", cp))
}
