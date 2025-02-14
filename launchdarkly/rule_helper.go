package launchdarkly

import (
	"errors"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go/v17"
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
				CONTEXT_KIND: {
					Type:             schema.TypeString,
					Optional:         true,
					Description:      "The context kind associated with the specified rollout. This argument is only valid if `rollout_weights` is also specified. Defaults to `user` if omitted.",
					DiffSuppressFunc: ruleContextKindDiffSuppressFunc(),
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

func ruleHasPercentageRollout(ruleMap map[string]interface{}) bool {
	return len(ruleMap[ROLLOUT_WEIGHTS].([]interface{})) > 0
}

func validateRuleResourceData(ruleMap map[string]interface{}) error {
	if !ruleHasPercentageRollout(ruleMap) {
		if ruleMap[BUCKET_BY].(string) != "" {
			return errors.New("rules: cannot use bucket_by argument with variation, only with rollout_weights")
		}
		if ruleMap[CONTEXT_KIND].(string) != "" {
			return errors.New("rules: cannot use context_kind argument with variation, only with rollout_weights")
		}
	}
	return nil
}

func ruleFromResourceData(val interface{}) (rule, error) {
	ruleMap := val.(map[string]interface{})
	err := validateRuleResourceData(ruleMap)
	if err != nil {
		return rule{}, err
	}
	var r rule
	for _, c := range ruleMap[CLAUSES].([]interface{}) {
		clause, err := clauseFromResourceData(c)
		if err != nil {
			return r, err
		}
		r.Clauses = append(r.Clauses, clause)
	}
	bucketBy := ruleMap[BUCKET_BY].(string)
	contextKind := ruleMap[CONTEXT_KIND].(string)
	rollout := rolloutFromResourceData(ruleMap[ROLLOUT_WEIGHTS].([]interface{}))
	if len(rollout.Variations) > 0 {
		r.Rollout = rollout
		if bucketBy != "" {
			r.Rollout.BucketBy = &bucketBy
		}
		if contextKind != "" {
			r.Rollout.ContextKind = &contextKind
		}
	} else {
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
			ruleMap[CONTEXT_KIND] = r.Rollout.ContextKind
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

func ruleContextKindDiffSuppressFunc() schema.SchemaDiffSuppressFunc {
	return func(k, oldValue, newValue string, d *schema.ResourceData) bool {
		if oldValue == "user" && newValue == "" {
			return true
		}
		return false
	}
}
