package launchdarkly

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
			CLIENT_SIDE_AVAILABILITY: {
				Type:     schema.TypeMap,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"using_environment_id": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"using_mobile_key": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},
			TAGS: tagsSchema(),
		},
	}
}

func dataSourceProjectRead(d *schema.ResourceData, meta interface{}) error {
	return projectRead(d, meta, true)
}
