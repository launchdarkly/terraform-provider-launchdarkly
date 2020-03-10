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
				"clauses": clauseSchema(),
				"weight": &schema.Schema{
					Type:         schema.TypeInt,
					Elem:         &schema.Schema{Type: schema.TypeInt},
					Optional:     true,
					ValidateFunc: validation.IntBetween(1, 100000),
				},
				"bucket_by": &schema.Schema{
					Type:         schema.TypeString,
					Elem:         &schema.Schema{Type: schema.TypeString},
					Optional:     true,
					ValidateFunc: validateKey(),
				},
			},
		},
	}
}

func segmentRulesFromResourceData(d *schema.ResourceData, metaRaw interface{}) []ldapi.UserSegmentRule {
	schemaRules := d.Get(RULES).([]interface{})
	rules := make([]ldapi.UserSegmentRule, len(schemaRules))
	for i, rule := range schemaRules {
		v := segmentRuleFromResourceData(rule)
		rules[i] = v
	}

	return rules
}

func segmentRuleFromResourceData(val interface{}) ldapi.UserSegmentRule {
	ruleMap := val.(map[string]interface{})
	r := ldapi.UserSegmentRule{
		Weight:   int32(ruleMap[WEIGHT].(int)),
		BucketBy: ruleMap[BUCKET_BY].(string),
	}
	for _, c := range ruleMap[CLAUSES].([]interface{}) {
		r.Clauses = append(r.Clauses, clauseFromResourceData(c))
	}

	return r
}

func segmentRulesToResourceData(rules []ldapi.UserSegmentRule) interface{} {
	transformed := make([]interface{}, len(rules))

	for i, r := range rules {
		transformed[i] = map[string]interface{}{
			CLAUSES:   clausesToResourceData(r.Clauses),
			WEIGHT:    r.Weight,
			BUCKET_BY: r.BucketBy,
		}
	}

	return transformed
}
