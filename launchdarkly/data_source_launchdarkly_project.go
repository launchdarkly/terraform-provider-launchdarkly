package launchdarkly

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceProject() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceProjectRead,

		Schema: map[string]*schema.Schema{
			KEY: {
				Type:     schema.TypeString,
				Required: true,
			},
			NAME: {
				Type:     schema.TypeString,
				Optional: true,
			},
			INCLUDE_IN_SNIPPET: {
				Type:     schema.TypeBool,
				Optional: true,
			},
			TAGS: tagsSchema(),
			ENVIRONMENTS: {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: environmentSchema(),
				},
			},
		},
	}
}

func dataSourceProjectRead(d *schema.ResourceData, meta interface{}) error {
	return projectRead(d, meta, true)
}
