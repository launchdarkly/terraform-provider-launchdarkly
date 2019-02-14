package launchdarkly

import (
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func tagsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Set:      stringSchemaSetFunc,
		Elem:     &schema.Schema{Type: schema.TypeString},
		Optional: true,
	}
}

func stringSetFromResourceData(d *schema.ResourceData, key string) []string {
	tags := d.Get(key).(*schema.Set)

	strs := make([]string, tags.Len())
	for i, tag := range tags.List() {
		strs[i] = tag.(string)
	}
	return strs
}

// https://godoc.org/github.com/hashicorp/terraform/helper/schema#SchemaSetFunc
func stringSchemaSetFunc(value interface{}) int {
	return hashcode.String(value.(string))
}
