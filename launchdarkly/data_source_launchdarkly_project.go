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
				Computed: true,
			},
			INCLUDE_IN_SNIPPET: {
				Type:     schema.TypeBool,
				Computed: true,
			},
			TAGS: tagsSchema(),
		},
	}
}

func dataSourceProjectRead(d *schema.ResourceData, meta interface{}) error {
	return projectRead(d, meta, true)
}
