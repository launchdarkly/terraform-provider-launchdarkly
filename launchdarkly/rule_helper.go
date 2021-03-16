package launchdarkly

import (
	"errors"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go"
)

func rulesSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				CLAUSES: clauseSchema(),
				VARIATION: {
					Type:         schema.TypeInt,
					Elem:         &schema.Schema{Type: schema.TypeInt},
					Optional:     true,
					ValidateFunc: validation.IntAtLeast(0),
				},
				ROLLOUT_WEIGHTS: rolloutSchema(),
				BUCKET_BY: {
					Type:     schema.TypeString,
					Optional: true,
				},
			},
		},
	}
}

type rule struct {
	Variation *int           `json:"variation,omitempty"`
	Rollout   *ldapi.Rollout `json:"rollout,omitempty"`
	Clauses   []ldapi.Clause `json:"clauses,omitempty"`
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
	bucketBy, bucketByFound := ruleMap["bucket_by"].(string)
	if len(rolloutFromResourceData(ruleMap[ROLLOUT_WEIGHTS]).Variations) > 0 {
		r.Rollout = rolloutFromResourceData(ruleMap[ROLLOUT_WEIGHTS])
		if bucketByFound {
			r.Rollout.BucketBy = bucketBy
		}
	} else {
		if bucketByFound && bucketBy != "" {
			return r, errors.New("rules: cannot use bucket_by argument with variation, only with rollout_weights")
		}
		r.Variation = intPtr(ruleMap[VARIATION].(int))
	}
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
		transformed = append(transformed, ruleMap)
	}
	return transformed, nil
}
