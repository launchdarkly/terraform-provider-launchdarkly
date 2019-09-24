package launchdarkly

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"

	ldapi "github.com/launchdarkly/api-client-go"
)

func fallthroughSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				rollout_weights: rolloutSchema(),
				variation: &schema.Schema{
					Type:         schema.TypeInt,
					Optional:     true,
					ValidateFunc: validation.IntAtLeast(0),
				},
			},
		},
	}
}

// fallthroughModel is used for patchReplace statements
type fallthroughModel struct {
	Variation *int           `json:"variation,omitempty"`
	Rollout   *ldapi.Rollout `json:"rollout,omitempty"`
}

func isPercentRollout(fall []interface{}) bool {
	for _, f := range fall {
		fallThrough := f.(map[string]interface{})
		if roll, ok := fallThrough[rollout_weights]; ok {
			return len(roll.([]interface{})) > 0
		}
	}
	return false
}

func fallthroughFromResourceData(d *schema.ResourceData) fallthroughModel {
	f := d.Get(flag_fallthrough).([]interface{})
	if len(f) == 0 {
		return fallthroughModel{Variation: intPtr(0)}
	}

	fall := f[0].(map[string]interface{})
	if isPercentRollout(f) {
		return fallthroughModel{Rollout: rolloutFromResourceData(fall[rollout_weights])}
	}
	val := fall[variation].(int)
	return fallthroughModel{Variation: &val}
}

func fallthroughToResourceData(fallThrough *ldapi.ModelFallthrough) interface{} {
	transformed := make([]interface{}, 1)
	if fallThrough.Rollout != nil {
		transformed[0] = map[string]interface{}{
			rollout_weights: rolloutsToResourceData(fallThrough.Rollout),
		}
	} else {
		transformed[0] = map[string]interface{}{
			variation: fallThrough.Variation,
		}
	}
	return transformed
}
