package launchdarkly

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go"
)

func segmentRulesSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				CLAUSES: clauseSchema(),
				WEIGHT: {
					Type:         schema.TypeInt,
					Elem:         &schema.Schema{Type: schema.TypeInt},
					Optional:     true,
					ValidateFunc: validation.IntBetween(1, 100000),
				},
				BUCKET_BY: {
					Type:         schema.TypeString,
					Elem:         &schema.Schema{Type: schema.TypeString},
					Optional:     true,
					ValidateFunc: validateKey(),
				},
			},
		},
	}
}

func segmentRulesFromResourceData(d *schema.ResourceData, metaRaw interface{}) ([]ldapi.UserSegmentRule, error) {
	schemaRules := d.Get(RULES).([]interface{})
	rules := make([]ldapi.UserSegmentRule, len(schemaRules))
	for i, rule := range schemaRules {
		v, err := segmentRuleFromResourceData(rule)
		if err != nil {
			return rules, err
		}
		rules[i] = v
	}

	return rules, nil
}

func segmentRuleFromResourceData(val interface{}) (ldapi.UserSegmentRule, error) {
	ruleMap := val.(map[string]interface{})
	r := ldapi.UserSegmentRule{
		Weight:   int32(ruleMap[WEIGHT].(int)),
		BucketBy: ruleMap[BUCKET_BY].(string),
	}
	for _, c := range ruleMap[CLAUSES].([]interface{}) {
		clause, err := clauseFromResourceData(c)
		if err != nil {
			return r, err
		}
		r.Clauses = append(r.Clauses, clause)
	}

	return r, nil
}

func segmentRulesToResourceData(rules []ldapi.UserSegmentRule) (interface{}, error) {
	transformed := make([]interface{}, len(rules))

	for i, r := range rules {
		clauses, err := clausesToResourceData(r.Clauses)
		if err != nil {
			return nil, err
		}
		transformed[i] = map[string]interface{}{
			CLAUSES:   clauses,
			WEIGHT:    r.Weight,
			BUCKET_BY: r.BucketBy,
		}
	}

	return transformed, nil
}
