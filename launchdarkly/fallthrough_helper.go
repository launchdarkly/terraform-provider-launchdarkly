package launchdarkly

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	ldapi "github.com/launchdarkly/api-client-go/v16"
)

// In the LD model, this corresponds to the VariationOrRollout type
func fallthroughSchema(forDataSource bool) *schema.Schema {
	elemSchema := map[string]*schema.Schema{
		ROLLOUT_WEIGHTS: rolloutSchema(),
		BUCKET_BY: {
			Type:        schema.TypeString,
			Optional:    !forDataSource,
			Computed:    forDataSource,
			Description: "Group percentage rollout by a custom attribute. This argument is only valid if rollout_weights is also specified.",
		},
		CONTEXT_KIND: {
			Type:             schema.TypeString,
			Optional:         !forDataSource, // the API will set this to "user" by default if it is not set
			Computed:         forDataSource,
			Description:      "The context kind associated with the specified rollout. This argument is only valid if rollout_weights is also specified. If omitted, defaults to `user`.",
			DiffSuppressFunc: fallthroughContextKindDiffSuppressFunc(),
		},
		VARIATION: {
			Type:             schema.TypeInt,
			Optional:         !forDataSource,
			Computed:         forDataSource,
			Description:      "The default integer variation index to serve if no `prerequisites`, `target`, or `rules` apply. You must specify either `variation` or `rollout_weights`.",
			ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(0)),
		},
	}

	if forDataSource {
		elemSchema = removeInvalidFieldsForDataSource(elemSchema)
	}

	return &schema.Schema{
		Type:        schema.TypeList,
		Required:    !forDataSource,
		Computed:    forDataSource,
		Description: "Nested block describing the default variation to serve if no `prerequisites`, `target`, or `rules` apply.",
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: elemSchema,
		},
	}
}

// fallthroughModel is used for patchReplace statements
type fallthroughModel struct {
	Variation *int           `json:"variation,omitempty"`
	Rollout   *ldapi.Rollout `json:"rollout,omitempty"`
}

func validateFallThroughResourceData(f []interface{}) error {
	for _, f := range f {
		if f == nil {
			return errors.New("feature flag fallthrough block cannot be empty. Please specify at least one of variation or rollout_weights")
		}
	}
	if !isPercentRollout(f) {
		fall := f[0].(map[string]interface{})
		if bucketBy, ok := fall[BUCKET_BY]; ok {
			if bucketBy.(string) != "" {
				return errors.New("flag fallthrough: cannot use bucket_by argument with variation, only with rollout_weights")
			}
		}
	}
	return nil
}

func isPercentRollout(fall []interface{}) bool {
	for _, f := range fall {
		fallThrough := f.(map[string]interface{})
		if roll, ok := fallThrough[ROLLOUT_WEIGHTS]; ok {
			return len(roll.([]interface{})) > 0
		}
	}
	return false
}

func fallthroughFromResourceData(d *schema.ResourceData) (fallthroughModel, error) {
	f := d.Get(FALLTHROUGH).([]interface{})
	err := validateFallThroughResourceData(f)
	if err != nil {
		return fallthroughModel{}, err
	}

	fall := f[0].(map[string]interface{})
	if isPercentRollout(f) {
		rollout := fallthroughModel{Rollout: rolloutFromResourceData(fall[ROLLOUT_WEIGHTS])}
		bucketBy := fall[BUCKET_BY].(string)
		if bucketBy != "" {
			rollout.Rollout.BucketBy = &bucketBy
		}
		contextKind := fall[CONTEXT_KIND].(string)
		if contextKind != "" {
			rollout.Rollout.ContextKind = &contextKind
		}
		return rollout, nil

	}
	val := fall[VARIATION].(int)
	return fallthroughModel{Variation: &val}, nil
}

func fallthroughToResourceData(fallThrough ldapi.VariationOrRolloutRep) interface{} {
	transformed := make([]interface{}, 1)
	if fallThrough.Rollout != nil {
		rollout := map[string]interface{}{
			ROLLOUT_WEIGHTS: rolloutsToResourceData(fallThrough.Rollout),
		}
		if fallThrough.Rollout.BucketBy != nil {
			rollout[BUCKET_BY] = fallThrough.Rollout.BucketBy
		}
		if fallThrough.Rollout.ContextKind != nil {
			rollout[CONTEXT_KIND] = fallThrough.Rollout.ContextKind
		}
		transformed[0] = rollout
	} else {
		transformed[0] = map[string]interface{}{
			VARIATION: fallThrough.Variation,
		}
	}
	return transformed
}

func fallthroughContextKindDiffSuppressFunc() schema.SchemaDiffSuppressFunc {
	return func(k, oldValue, newValue string, d *schema.ResourceData) bool {
		if oldValue == "user" && newValue == "" {
			return true
		}
		return false
	}
}
