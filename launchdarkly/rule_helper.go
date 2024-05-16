package launchdarkly

import (
	"errors"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go/v16"
)

func rulesSchema(isDataSource bool) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    !isDataSource,
		Computed:    isDataSource,
		Description: "List of logical targeting rules.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				DESCRIPTION: {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "A human-readable description of the targeting rule.",
				},
				CLAUSES: clauseSchema(),
				VARIATION: {
					Type:             schema.TypeInt,
					Elem:             &schema.Schema{Type: schema.TypeInt},
					Optional:         true,
					Description:      "The integer variation index to serve if the rule clauses evaluate to `true`. You must specify either `variation` or `rollout_weights`",
					ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(0)),
				},
				ROLLOUT_WEIGHTS: rolloutSchema(),
				BUCKET_BY: {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Group percentage rollout by a custom attribute. This argument is only valid if `rollout_weights` is also specified.",
				},
			},
		},
	}
}

type rule struct {
	Description *string        `json:"description,omitempty"`
	Variation   *int           `json:"variation,omitempty"`
	Rollout     *ldapi.Rollout `json:"rollout,omitempty"`
	Clauses     []ldapi.Clause `json:"clauses,omitempty"`
}

func rulesFromResourceData(d *schema.ResourceData) ([]rule, error) {
	schemaRules := d.Get(RULES).([]interface{})
	rules := make([]rule, 0, len(schemaRules))
	for _, r := range schemaRules {
		rule, err := ruleFromResourceData(r)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

func ruleFromResourceData(val interface{}) (rule, error) {
	ruleMap := val.(map[string]interface{})
	var r rule
	for _, c := range ruleMap[CLAUSES].([]interface{}) {
		clause, err := clauseFromResourceData(c)
		if err != nil {
			return r, err
		}
		r.Clauses = append(r.Clauses, clause)
	}
	bucketBy := ruleMap[BUCKET_BY].(string)
	bucketByFound := bucketBy != ""
	if len(rolloutFromResourceData(ruleMap[ROLLOUT_WEIGHTS]).Variations) > 0 {
		r.Rollout = rolloutFromResourceData(ruleMap[ROLLOUT_WEIGHTS])
		if bucketByFound {
			r.Rollout.BucketBy = &bucketBy
		}
	} else {
		if bucketByFound && bucketBy != "" {
			return r, errors.New("rules: cannot use bucket_by argument with variation, only with rollout_weights")
		}
		r.Variation = intPtr(ruleMap[VARIATION].(int))
	}
	description := ruleMap[DESCRIPTION].(string)
	r.Description = &description
	log.Printf("[DEBUG] %+v\n", r)
	return r, nil
}

func rulesToResourceData(rules []ldapi.Rule) (interface{}, error) {
	transformed := make([]interface{}, 0, len(rules))

	for _, r := range rules {
		ruleMap := make(map[string]interface{})
		if len(r.Clauses) > 0 {
			clauses, err := clausesToResourceData(r.Clauses)
			if err != nil {
				return nil, err
			}
			ruleMap[CLAUSES] = clauses
		}
		if r.Rollout != nil {
			ruleMap[ROLLOUT_WEIGHTS] = rolloutsToResourceData(r.Rollout)
			ruleMap[BUCKET_BY] = r.Rollout.BucketBy
		} else {
			ruleMap[VARIATION] = r.Variation
		}
		if r.Description != nil {
			ruleMap[DESCRIPTION] = r.GetDescription()
		}
		transformed = append(transformed, ruleMap)
	}
	return transformed, nil
}
