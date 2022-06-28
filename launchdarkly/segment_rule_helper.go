package launchdarkly

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go/v10"
)

func segmentRulesSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				CLAUSES: clauseSchema(),
				WEIGHT: {
					Type:             schema.TypeInt,
					Elem:             &schema.Schema{Type: schema.TypeInt},
					Optional:         true,
					ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(1, 100000)),
					Description:      "The integer weight of the rule (between 1 and 100000).",
				},
				BUCKET_BY: {
					Type:        schema.TypeString,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Optional:    true,
					Description: "The attribute by which to group users together.",
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
	rawClauses := ruleMap[CLAUSES].([]interface{})
	clauses := make([]ldapi.Clause, 0, len(rawClauses))
	for _, c := range rawClauses {
		clause, err := clauseFromResourceData(c)
		if err != nil {
			return ldapi.UserSegmentRule{}, err
		}
		clauses = append(clauses, clause)
	}

	r := ldapi.NewUserSegmentRule(clauses)

	bucketBy := ruleMap[BUCKET_BY].(string)
	if bucketBy != "" {
		r.SetBucketBy(bucketBy)
	}

	// weight == 0 when omitted from the config. In this case we do not want to include a weight in the PATCH
	// request because the results will be counterintuitive. See: https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/79
	weight := int32(ruleMap[WEIGHT].(int))
	if weight > 0 {
		r.SetWeight(weight)
	}

	return *r, nil
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
