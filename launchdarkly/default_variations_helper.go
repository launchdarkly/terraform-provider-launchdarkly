package launchdarkly

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v12"
)

func defaultVariationsFromResourceData(d *schema.ResourceData) (*ldapi.Defaults, error) {
	schemaVariations := d.Get(VARIATIONS).([]interface{})
	numberOfVariations := len(schemaVariations)
	variationType := d.Get(VARIATION_TYPE).(string)
	rawDefaults, ok := d.GetOk(DEFAULTS)
	if !ok {
		defaultOff := numberOfVariations - 1
		if variationType == BOOL_VARIATION {
			defaultOff = 1 // otherwise at this point it would be -1
		}
		return &ldapi.Defaults{OnVariation: int32(0), OffVariation: int32(defaultOff)}, nil
	}
	defaultList := rawDefaults.([]interface{})
	if variationType == BOOL_VARIATION {
		if numberOfVariations == 0 && len(defaultList) == 0 {
			// default boolean variations
			return &ldapi.Defaults{OnVariation: int32(0), OffVariation: int32(1)}, nil
		} else {
			// this allows us to confidence check the variation indices below
			numberOfVariations = 2
		}
	}

	defaults := defaultList[0].(map[string]interface{})
	on := defaults[ON_VARIATION].(int)
	off := defaults[OFF_VARIATION].(int)

	if on >= numberOfVariations {
		return nil, fmt.Errorf("default on_variation %v is out of range, must be between 0 and %v inclusive", on, numberOfVariations-1)
	}
	if off >= numberOfVariations {
		return nil, fmt.Errorf("default off_variation %v is out of range, must be between 0 and %v inclusive", off, numberOfVariations-1)
	}

	return &ldapi.Defaults{OnVariation: int32(on), OffVariation: int32(off)}, nil
}
