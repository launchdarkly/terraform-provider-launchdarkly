package launchdarkly

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"

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
				bucket_by: {
					Type:     schema.TypeString,
					Optional: true,
				},
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

func validateFallThroughResourceData(f []interface{}) error {
	if len(f) == 0 {
		return nil
	}

	if !isPercentRollout(f) {
		fall := f[0].(map[string]interface{})
		if bucketBy, ok := fall[bucket_by]; ok {
			if bucketBy.(string) != "" {
				return errors.New("flag_fallthrough: cannot use bucket_by argument with variation, only with rollout_weights")
			}
		}
	}
	return nil
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

func fallthroughFromResourceData(d *schema.ResourceData) (fallthroughModel, error) {
	f := d.Get(flag_fallthrough).([]interface{})
	err := validateFallThroughResourceData(f)
	if err != nil {
		return fallthroughModel{}, err
	}

	if len(f) == 0 {
		return fallthroughModel{Variation: intPtr(0)}, nil
	}

	fall := f[0].(map[string]interface{})
	if isPercentRollout(f) {
		rollout := fallthroughModel{Rollout: rolloutFromResourceData(fall[rollout_weights])}
		bucketBy, ok := fall[bucket_by]
		if ok {
			rollout.Rollout.BucketBy = bucketBy.(string)
		}
		return rollout, nil

	}
	val := fall[variation].(int)
	return fallthroughModel{Variation: &val}, nil
}

func fallthroughToResourceData(fallThrough *ldapi.ModelFallthrough) interface{} {
	transformed := make([]interface{}, 1)
	if fallThrough.Rollout != nil {
		rollout := map[string]interface{}{
			rollout_weights: rolloutsToResourceData(fallThrough.Rollout),
		}
		if fallThrough.Rollout.BucketBy != "" {
			rollout[bucket_by] = fallThrough.Rollout.BucketBy
		}
		transformed[0] = rollout
	} else {
		transformed[0] = map[string]interface{}{
			variation: fallThrough.Variation,
		}
	}
	return transformed
}
