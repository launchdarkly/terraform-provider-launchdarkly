package launchdarkly

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func tagsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Set:      schema.HashString,
		Elem:     &schema.Schema{Type: schema.TypeString},
		Optional: true,
	}
}

func stringsFromResourceData(d *schema.ResourceData, key string) []string {
	return stringsFromSchemaSet(d.Get(key).(*schema.Set))
}

func stringsFromSchemaSet(schemaSet *schema.Set) []string {
	strs := make([]string, schemaSet.Len())
	for i, tag := range schemaSet.List() {
		strs[i] = tag.(string)
	}
	return strs
}
