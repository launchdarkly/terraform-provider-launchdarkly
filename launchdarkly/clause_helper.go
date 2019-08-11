package launchdarkly

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"

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

// https://godoc.org/github.com/hashicorp/terraform/helper/schema#SchemaSetFunc
func clauseHash(val interface{}) int {
	clause := clauseFromResourceData(val)
	return hashcode.String(fmt.Sprintf("%v", clause))
}

func clauseFromResourceData(val interface{}) ldapi.Clause {
	clauseMap := val.(map[string]interface{})
	c := ldapi.Clause{
		Attribute: clauseMap[attribute].(string),
		Op:        clauseMap[op].(string),
		Negate:    clauseMap[negate].(bool),
	}
	for _, v := range clauseMap[values].([]interface{}) {
		c.Values = append(c.Values, v)
	}

	return c
}

func clausesToResourceData(clauses []ldapi.Clause) interface{} {
	transformed := make([]interface{}, len(clauses))

	for i, c := range clauses {
		transformed[i] = map[string]interface{}{
			attribute: c.Attribute,
			op:        c.Op,
			values:    c.Values,
			negate:    c.Negate,
		}
	}
	return transformed
}
