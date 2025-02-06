package launchdarkly

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func roleAttributesSchema(isDataSource bool) *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				KEY: {
					Type:        schema.TypeString,
					Required:    true,
					Description: "The key / name of your role attribute. In the example `$${roleAttribute/testAttribute}`, the key is `testAttribute`.",
				},
				VALUES: {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Required:    true,
					Description: "A list of values for your role attribute. For example, if your policy statement defines the resource `\"proj/$${roleAttribute/testAttribute}\"`, the values would be the keys of the projects you wanted to assign access to.",
				},
			},
		},
		Optional:    true,
		Computed:    isDataSource,
		Description: "A role attributes block. One block must be defined per role attribute. The key is the role attribute key and the value is a string array of resource keys that apply.",
	}
}
