package launchdarkly

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go/v7"
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
		ValidateDiagFunc: validation.ToDiagFunc(validateVariationType),
	}
}

func variationsSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		Computed:    true,
		Description: "An array of possible variations for the flag",
		MinItems:    2,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				NAME: {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "A name for the variation",
				},
				DESCRIPTION: {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "A description for the variation",
				},
				VALUE: {
					Type:             schema.TypeString,
					Required:         true,
					Description:      "The value of the flag for this variation",
					ValidateDiagFunc: validation.ToDiagFunc(validateVariationValue),
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
		errs = append(errs, fmt.Errorf("%q contains an invalid value %q. Valid values are `boolean`, `string`, `number`, and `json`", key, value))
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
	variationType := d.Get(VARIATION_TYPE).(string)
	old, new := d.GetChange(VARIATIONS)

	if len(old.([]interface{})) == 0 {
		// This can only happen when the resource is first created. Since this is handled in the creation POST,
		// variation patches are not necessary.
		return patches, nil
	}

	oldVariations, err := variationsFromSchemaData(old, variationType)
	if err != nil {
		return patches, err
	}

	newVariations, err := variationsFromSchemaData(new, variationType)
	if err != nil {
		return patches, err
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
	if variationType != BOOL_VARIATION && len(list) < 2 {
		return variations, fmt.Errorf("multivariate flags must have at least two variations defined")
	}

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
	schemaVariations := d.Get(VARIATIONS)
	variationType := d.Get(VARIATION_TYPE).(string)
	variations, err := variationsFromSchemaData(schemaVariations, variationType)
	if err != nil {
		return variations, err
	}
	return variations, nil
}

func boolVariationFromResourceData(variation interface{}) ldapi.Variation {
	variationMap := variation.(map[string]interface{})
	v := variationMap[VALUE].(string) == "true"
	transformed := ldapi.Variation{
		Value: ptr(v),
	}
	name := variationMap[NAME].(string)
	if name != "" {
		transformed.Name = &name
	}
	description := variationMap[DESCRIPTION].(string)
	if description != "" {
		transformed.Description = &description
	}
	return transformed
}

func stringVariationFromResourceData(variation interface{}) ldapi.Variation {
	var transformed ldapi.Variation
	if variation == nil { // handle empty string value
		transformed.Value = strPtr("")
		return transformed
	}
	variationMap := variation.(map[string]interface{})
	v := variationMap[VALUE]
	transformed.Value = &v
	name := variationMap[NAME].(string)
	if name != "" {
		transformed.Name = &name
	}
	description := variationMap[DESCRIPTION].(string)
	if description != "" {
		transformed.Description = &description
	}
	return transformed
}

func numberVariationFromResourceData(variation interface{}) (ldapi.Variation, error) {
	variationMap := variation.(map[string]interface{})
	stringValue := variationMap[VALUE].(string)
	v, err := strconv.ParseFloat(stringValue, 64)
	if err != nil {
		return ldapi.Variation{}, fmt.Errorf("%q is an invalid number variation value. %v", stringValue, err)
	}
	transformed := ldapi.Variation{Value: ptr(v)}
	name := variationMap[NAME].(string)
	if name != "" {
		transformed.Name = &name
	}
	description := variationMap[DESCRIPTION].(string)
	if description != "" {
		transformed.Description = &description
	}
	return transformed, nil
}

func jsonVariationFromResourceData(variation interface{}) (ldapi.Variation, error) {
	variationMap := variation.(map[string]interface{})
	stringValue := variationMap[VALUE].(string)
	var v interface{}
	err := json.Unmarshal([]byte(stringValue), &v)
	if err != nil {
		return ldapi.Variation{}, fmt.Errorf("%q is an invalid json variation value. %v", stringValue, err)
	}
	transformed := ldapi.Variation{Value: ptr(v)}
	name := variationMap[NAME].(string)
	if name != "" {
		transformed.Name = &name
	}
	description := variationMap[DESCRIPTION].(string)
	if description != "" {
		transformed.Description = &description
	}
	return transformed, nil
}

func stringifyValue(value interface{}) string {
	var str string
	switch v := (value).(type) {
	case int:
		str = strconv.Itoa(v)
	case float64:
		str = strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		str = strconv.FormatBool(v)
	case string:
		str = v
	}
	return str
}

func variationValueToString(value *interface{}, variationType string) (string, error) {
	if variationType != JSON_VARIATION {
		return stringifyValue(*value), nil
	}
	byteVal, err := json.Marshal(*value)
	if err != nil {
		return "", fmt.Errorf("unable to marshal json variation value: %v", err)
	}
	ret, err := structure.NormalizeJsonString(string(byteVal))
	if err != nil {
		return "", fmt.Errorf("unable to normalize json variation value: %v", err)
	}
	return ret, nil
}

func variationsToResourceData(variations []ldapi.Variation, variationType string) (interface{}, error) {
	transformed := make([]interface{}, 0, len(variations))

	for _, variation := range variations {
		v, err := variationValueToString(&variation.Value, variationType)
		if err != nil {
			return nil, err
		}

		transformed = append(transformed, map[string]interface{}{
			NAME:        variation.Name,
			DESCRIPTION: variation.Description,
			VALUE:       v,
		})
	}
	return transformed, nil
}

func variationsToVariationType(variations []ldapi.Variation) (string, error) {
	// since all variations have a uniform type, checking the first variation is sufficient
	variationValue := variations[0].Value
	if variationValue == nil {
		return "", fmt.Errorf("nil variation value: %v", variationValue)
	}
	var variationType string
	switch variationValue.(type) {
	case bool:
		variationType = BOOL_VARIATION
	case string:
		variationType = STRING_VARIATION
	case float64:
		variationType = NUMBER_VARIATION
	case map[string]interface{}, []interface{}:
		variationType = JSON_VARIATION
	default:
		return "", fmt.Errorf("unknown variation type: %q", reflect.TypeOf(variationValue))
	}
	return variationType, nil
}
