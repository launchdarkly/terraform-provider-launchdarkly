package launchdarkly

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type tagsSchemaOptions struct {
	isDataSource bool
}

func tagsSchema(options tagsSchemaOptions) *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeSet,
		Set:  schema.HashString,
		Elem: &schema.Schema{
			Type: schema.TypeString,
			// Can't use validation.ToDiagFunc converted validators on TypeSet at the moment
			// https://github.com/hashicorp/terraform-plugin-sdk/issues/734
			ValidateFunc: validateTagsNoDiag(),
		},
		Optional:    !options.isDataSource,
		Computed:    options.isDataSource,
		Description: "Tags associated with your resource.",
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
