package launchdarkly

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
)

func clauseSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"attribute": {
					Type:     schema.TypeString,
					Required: true,
				},
				"op": {
					Type:         schema.TypeString,
					Required:     true,
					ValidateFunc: validateOp(),
				},
				"values": {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Required: true,
				},
				"negate": {
					Type:     schema.TypeBool,
					Required: true,
				},
			},
		},
	}
}

func clauseFromResourceData(val interface{}) ldapi.Clause {
	clauseMap := val.(map[string]interface{})
	c := ldapi.Clause{
		Attribute: clauseMap[ATTRIBUTE].(string),
		Op:        clauseMap[OP].(string),
		Negate:    clauseMap[NEGATE].(bool),
	}
	c.Values = append(c.Values, clauseMap[VALUES].([]interface{})...)
	return c
}

func clausesToResourceData(clauses []ldapi.Clause) interface{} {
	transformed := make([]interface{}, len(clauses))

	for i, c := range clauses {
		transformed[i] = map[string]interface{}{
			ATTRIBUTE: c.Attribute,
			OP:        c.Op,
			VALUES:    c.Values,
			NEGATE:    c.Negate,
		}
	}
	return transformed
}
