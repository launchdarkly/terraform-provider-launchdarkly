package launchdarkly

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/structure"
	ldapi "github.com/launchdarkly/api-client-go"
)

const (
	BOOL_VARIATION   = "boolean"
	STRING_VARIATION = "string"
	NUMBER_VARIATION = "number"
	JSON_VARIATION   = "json"
)

func variationTypeSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeString,
		Required: true,
		ForceNew: true,
		Description: fmt.Sprintf("The uniform type for all variations. Can be either %q, %q, %q, or %q.",
			BOOL_VARIATION, STRING_VARIATION, NUMBER_VARIATION, JSON_VARIATION),
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
					Type:         schema.TypeString,
					Required:     true,
					ValidateFunc: validateVariationValue,
					StateFunc: func(i interface{}) string {
						// All values are stored as strings in TF state
						v, err := structure.NormalizeJsonString(i)
						if err != nil {
							return fmt.Sprintf("%v", i)
						}
						return v
					},
				},
			},
		},
	}
}

func validateVariationType(val interface{}, key string) (warns []string, errs []error) {
	value := val.(string)
	switch value {
	case BOOL_VARIATION, STRING_VARIATION, NUMBER_VARIATION, JSON_VARIATION:
		break
	default:
		errs = append(errs, fmt.Errorf("%q contains an invalid value %q. Valid values are `boolean` and `string`", key, value))
	}
	return warns, errs
}

func validateVariationValue(val interface{}, key string) (warns []string, errs []error) {
	value := strings.TrimSpace(val.(string))
	if strings.HasPrefix(value, "{") {
		if !json.Valid([]byte(value)) {
			warns = append(warns, fmt.Sprintf("%q starts with a '{' but is not valid JSON. received: %q", key, value))
		}
	}
	return warns, errs
}

func variationPatchesFromResourceData(d *schema.ResourceData) ([]ldapi.PatchOperation, error) {
	var patches []ldapi.PatchOperation
	variationType := d.Get(variation_type).(string)
	old, new := d.GetChange(variations)

	oldVariations, err := variationsFromSchemaData(old, variationType)
	if err != nil {
		return patches, err
	}

	newVariations, err := variationsFromSchemaData(new, variationType)
	if err != nil {
		return patches, err
	}

	if len(oldVariations) == 0 {
		// This can only happen when the resource is first created. Since this is handled in the creation POST,
		// variation patches are not necessary.
		return patches, nil
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
	return patches, nil
}

func variationsFromSchemaData(schemaVariations interface{}, variationType string) ([]ldapi.Variation, error) {
	list := schemaVariations.([]interface{})
	variations := make([]ldapi.Variation, len(list))

	var err error
	for i, variation := range list {
		switch variationType {
		case BOOL_VARIATION:
			variations[i] = boolVariationFromResourceData(variation)
		case STRING_VARIATION:
			variations[i] = stringVariationFromResourceData(variation)
		case NUMBER_VARIATION:
			variations[i], err = numberVariationFromResourceData(variation)
		case JSON_VARIATION:
			variations[i], err = jsonVariationFromResourceData(variation)
		default:
			return variations, fmt.Errorf("invalid variation type: %q", variationType)
		}
		if err != nil {
			return variations, err
		}
	}
	return variations, nil
}

func variationsFromResourceData(d *schema.ResourceData) ([]ldapi.Variation, error) {
	schemaVariations := d.Get(variations)
	variationType := d.Get(variation_type).(string)
	variations, err := variationsFromSchemaData(schemaVariations, variationType)
	if err != nil {
		return variations, err
	}
	return variations, nil
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

func numberVariationFromResourceData(variation interface{}) (ldapi.Variation, error) {
	variationMap := variation.(map[string]interface{})
	stringValue := variationMap[value].(string)
	v, err := strconv.ParseFloat(stringValue, 64)
	if err != nil {
		return ldapi.Variation{}, fmt.Errorf("%q is an invalid number variation value. %v", stringValue, err)
	}
	return ldapi.Variation{
		Name:        variationMap[name].(string),
		Description: variationMap[description].(string),
		Value:       ptr(v),
	}, nil
}

func jsonVariationFromResourceData(variation interface{}) (ldapi.Variation, error) {
	variationMap := variation.(map[string]interface{})
	stringValue := variationMap[value].(string)
	var v map[string]interface{}
	err := json.Unmarshal([]byte(stringValue), &v)
	if err != nil {
		return ldapi.Variation{}, fmt.Errorf("%q is an invalid json variation value. %v", stringValue, err)
	}
	return ldapi.Variation{
		Name:        variationMap[name].(string),
		Description: variationMap[description].(string),
		Value:       ptr(v),
	}, nil
}

func variationsToResourceData(variations []ldapi.Variation, variationType string) (interface{}, error) {
	transformed := make([]interface{}, 0, len(variations))

	for _, variation := range variations {
		var v string
		if variationType != JSON_VARIATION {
			v = fmt.Sprintf("%v", *variation.Value)
		} else {
			byteVal, err := json.Marshal(*variation.Value)
			if err != nil {
				return transformed, fmt.Errorf("unable to marshal json variation: %v", err)
			}
			v, err = structure.NormalizeJsonString(string(byteVal))
			if err != nil {
				return transformed, fmt.Errorf("unable to normalize json variation: %v", err)
			}
		}

		transformed = append(transformed, map[string]interface{}{
			name:        variation.Name,
			description: variation.Description,
			value:       v,
		})
	}
	return transformed, nil
}

func variationsToVariationType(variations []ldapi.Variation) (string, error) {
	// since all variations have a uniform type, checking the first variation is sufficient
	valPtr := variations[0].Value
	if valPtr == nil {
		return "", fmt.Errorf("nil variation value: %v", valPtr)
	}
	variationValue := *valPtr
	var variationType string
	switch variationValue.(type) {
	case bool:
		variationType = BOOL_VARIATION
	case string:
		variationType = STRING_VARIATION
	case float64:
		variationType = NUMBER_VARIATION
	case map[string]interface{}:
		variationType = JSON_VARIATION
	default:
		return "", fmt.Errorf("unknown variation type: %q", reflect.TypeOf(variationValue))
	}
	return variationType, nil
}
