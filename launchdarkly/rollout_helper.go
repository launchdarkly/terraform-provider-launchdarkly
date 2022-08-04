package launchdarkly

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	ldapi "github.com/launchdarkly/api-client-go/v10"
)

func rolloutSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Schema{
			Type: schema.TypeInt,
			// Can't use validation.ToDiagFunc converted validators on TypeList at the moment
			// https://github.com/hashicorp/terraform-plugin-sdk/issues/734
			ValidateFunc: validation.IntBetween(0, 100000),
		},
	}
}

func rolloutFromResourceData(val interface{}) *ldapi.Rollout {
	rolloutList := val.([]interface{})
	variations := []ldapi.WeightedVariation{}
	for idx, k := range rolloutList {
		weight := k.(int)
		variations = append(variations,
			ldapi.WeightedVariation{
				Variation: int32(idx),
				Weight:    int32(weight),
			})
	}

	r := ldapi.Rollout{
		Variations: variations,
	}
	log.Printf("[DEBUG] %+v\n", r)

	return &r
}

func rolloutsToResourceData(rollouts *ldapi.Rollout) interface{} {
	transformed := make([]interface{}, len(rollouts.Variations))

	for _, r := range rollouts.Variations {
		transformed[r.Variation] = r.Weight
	}
	return transformed
}
