package launchdarkly

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func roleAttributesSchema(isDataSource bool) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeMap,
		Optional:    true,
		Computed:    isDataSource,
		Description: "A map of role attributes. The key is the role attribute key and the value is a string array of resource keys that apply.",
	}
}
