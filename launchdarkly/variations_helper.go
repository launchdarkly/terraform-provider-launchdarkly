package launchdarkly

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
)

const (
	BOOL_VARIATION   = "boolean"
	STRING_VARIATION = "string"
)

func variationTypeSchema() *schema.Schema {
	return &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		Description:  "The uniform type for all variations. Can be either `boolean` or `string`.",
		ValidateFunc: validateVariationType,
	}
}

func variationsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Computed: true,
		MinItems: 2,
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
						// This will work for bool and string variation_types
						return fmt.Sprintf("%v", i)
					},
				},
			},
		},
	}
}

func validateVariationType(val interface{}, key string) (warns []string, errs []error) {
	value := val.(string)
	switch value {
	// TODO: add Number and JSON
	case BOOL_VARIATION, STRING_VARIATION:
		break
	default:
		errs = append(errs, fmt.Errorf("%q contains an invalid value %q. Valid values are `boolean` and `string`", key, value))
	}
	return warns, errs
}

func variationPatchesFromResourceData(d *schema.ResourceData) []ldapi.PatchOperation {
	var patches []ldapi.PatchOperation
	variationType := d.Get(variation_type).(string)
	old, new := d.GetChange(variations)
	oldVariations := variationsFromSchemaData(old, variationType)
	newVariations := variationsFromSchemaData(new, variationType)

	if len(oldVariations) == 0 {
		// This can only happen when the resource is first created. Since this is handled in the creation POSTm
		// variation patches are not necessary.
		return patches
	}

	// remove any unnecessary variations from the end of the variation slice
	for idx := len(newVariations); idx < len(oldVariations); idx++ {
		patches = append(patches, patchRemove(fmt.Sprintf("/variations/%d", idx)))
	}

	for idx, variation := range newVariations {
		if idx < len(oldVariations) {
			patches = append(patches, patchReplace(fmt.Sprintf("/variations/%d/value", idx), variation.Value))
			patches = append(patches, patchReplace(fmt.Sprintf("/variations/%d/name", idx), variation.Name))
			patches = append(patches, patchReplace(fmt.Sprintf("/variations/%d/description", idx), variation.Description))
		} else {
			patches = append(patches, patchAdd(fmt.Sprintf("/variations/%d", idx), variation))
		}
	}
	return patches
}

func variationsFromSchemaData(schemaVariations interface{}, variationType string) []ldapi.Variation {
	list := schemaVariations.([]interface{})
	variations := make([]ldapi.Variation, len(list))

	for i, variation := range list {
		switch variationType {
		case "boolean":
			variations[i] = boolVariationFromResourceData(variation)
		case "string":
			variations[i] = stringVariationFromResourceData(variation)
		default:
			variations[i] = boolVariationFromResourceData(variation)
		}
	}
	return variations
}

func variationsFromResourceData(d *schema.ResourceData) []ldapi.Variation {
	schemaVariations := d.Get(variations)
	variationType := d.Get(variation_type).(string)
	return variationsFromSchemaData(schemaVariations, variationType)
}

func boolVariationFromResourceData(variation interface{}) ldapi.Variation {
	variationMap := variation.(map[string]interface{})
	v := variationMap[value].(string) == "true"
	return ldapi.Variation{
		Name:        variationMap[name].(string),
		Description: variationMap[description].(string),
		Value:       ptr(v),
	}
}

func stringVariationFromResourceData(variation interface{}) ldapi.Variation {
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

func variationsToVariationType(variations []ldapi.Variation) string {
	// since all variations have a uniform type, checking the first variation is sufficient
	variationValue := *variations[0].Value
	var variationType string
	switch variationValue.(type) {
	case bool:
		variationType = BOOL_VARIATION
	case string:
		variationType = STRING_VARIATION
	default:
		variationType = BOOL_VARIATION
	}
	return variationType
}
