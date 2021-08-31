package launchdarkly

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
)

func defaultVariationsFromResourceData(d *schema.ResourceData) (*ldapi.Defaults, error) {
	schemaVariations := d.Get(VARIATIONS).([]interface{})
	variationType := d.Get(VARIATION_TYPE).(string)
	if len(schemaVariations) == 0 && variationType == BOOL_VARIATION {
		// default boolean variations
		return &ldapi.Defaults{OnVariation: int32(0), OffVariation: int32(1)}, nil
	}
	rawDefaults, ok := d.GetOk(DEFAULTS)
	if !ok {
		return &ldapi.Defaults{OnVariation: int32(0), OffVariation: int32(len(schemaVariations) - 1)}, nil
	}
	defaultList := rawDefaults.([]interface{})
	defaults := defaultList[0].(map[string]interface{})
	on := defaults[ON_VARIATION].(int)
	off := defaults[OFF_VARIATION].(int)

	if on >= len(schemaVariations) {
		return nil, fmt.Errorf("default on_variation %v is out of range, must be between 0 and %v inclusive", on, len(schemaVariations)-1)
	}
	if off >= len(schemaVariations) {
		return nil, fmt.Errorf("default off_variation %v is out of range, must be between 0 and %v inclusive", off, len(schemaVariations)-1)
	}

	return &ldapi.Defaults{OnVariation: int32(on), OffVariation: int32(off)}, nil
}
