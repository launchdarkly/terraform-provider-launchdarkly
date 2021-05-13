package launchdarkly

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
)

var defaultBooleanVariations = []interface{}{
	map[string]interface{}{
		VALUE: "true",
	},
	map[string]interface{}{
		VALUE: "false",
	},
}

var errDefaultsNotSet = errors.New("default variations not set")

func validateDefaultVariations(d *schema.ResourceData) error {
	schemaVariations := d.Get(VARIATIONS).([]interface{})
	onValue, onOk := d.GetOk(DEFAULT_ON_VARIATION)
	offValue, offOk := d.GetOk(DEFAULT_OFF_VARIATION)

	if !onOk && !offOk {
		return errDefaultsNotSet
	}
	if onOk && !offOk {
		return fmt.Errorf("default_off_variation is required when default_on_variation is defined")
	}
	if !onOk && offOk {
		return fmt.Errorf("default_on_variation is required when default_off_variation is defined")
	}

	if len(schemaVariations) == 0 {
		schemaVariations = defaultBooleanVariations
	}

	onFound := false
	offFound := false
	for _, v := range schemaVariations {
		variation := v.(map[string]interface{})
		if variation[VALUE].(string) == onValue.(string) {
			onFound = true
		}
		if variation[VALUE].(string) == offValue.(string) {
			offFound = true
		}
	}
	if !onFound {
		return fmt.Errorf("default_on_variation %q is not defined as a variation", onValue)
	}
	if !offFound {
		return fmt.Errorf("default_off_variation %q is not defined as a variation", offValue)
	}
	return nil
}

func defaultVariationsFromResourceData(d *schema.ResourceData) (*ldapi.Defaults, error) {
	err := validateDefaultVariations(d)
	if err != nil {
		if err == errDefaultsNotSet {
			return nil, nil
		}
		return nil, err
	}
	schemaVariations := d.Get(VARIATIONS).([]interface{})
	if len(schemaVariations) == 0 {
		schemaVariations = defaultBooleanVariations
	}
	onValue := d.Get(DEFAULT_ON_VARIATION).(string)
	offValue := d.Get(DEFAULT_OFF_VARIATION).(string)

	var on *int
	var off *int
	for i, v := range schemaVariations {
		i := i
		variation := v.(map[string]interface{})
		val := variation[VALUE].(string)
		if val == onValue {
			on = &i
		}
		if val == offValue {
			off = &i
		}
	}

	return &ldapi.Defaults{OnVariation: int32(*on), OffVariation: int32(*off)}, nil
}
