package launchdarkly

import (
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func tagsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Set:      stringHash,
		Elem:     &schema.Schema{Type: schema.TypeString},
		Optional: true,
	}
}

func stringSetFromResourceData(d *schema.ResourceData, key string) []string {
	return stringSetFromSchemaSet(d.Get(key).(*schema.Set))
}

func stringSetFromSchemaSet(schemaSet *schema.Set) []string {
	strs := make([]string, schemaSet.Len())
	for i, tag := range schemaSet.List() {
		strs[i] = tag.(string)
	}
	return strs
}

// https://godoc.org/github.com/hashicorp/terraform/helper/schema#SchemaSetFunc
func stringHash(value interface{}) int {
	return hashcode.String(value.(string))
}
