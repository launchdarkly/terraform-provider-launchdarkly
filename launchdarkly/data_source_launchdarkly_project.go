package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceProject() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceProjectRead,

		Description: "Provides a LaunchDarkly project data source.\n\nThis data source allows you to retrieve project information from your LaunchDarkly organization.\n\n-> **Note:** LaunchDarkly data sources do not provide access to the project's environments. If you wish to import environment configurations as data sources you must use the [`launchdarkly_environment` data source](/docs/providers/launchdarkly/d/environment.html).",

		Schema: map[string]*schema.Schema{
			KEY: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The project's unique key.",
			},
			NAME: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The project's name.",
			},
			CLIENT_SIDE_AVAILABILITY: {
				Type:        schema.TypeList,
				Computed:    true,
				Deprecated:  "'client_side_availability' is now deprecated. Please migrate to 'default_client_side_availability' to maintain future compatibility.",
				Description: "A map describing which client-side SDKs can use new flags by default. Please migrate to `default_client_side_availability` to maintain future compatibility.",
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
			DEFAULT_CLIENT_SIDE_AVAILABILITY: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "A block describing which client-side SDKs can use new flags by default.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"using_environment_id": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"using_mobile_key": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
			TAGS: tagsSchema(tagsSchemaOptions{isDataSource: true}),
		},
	}
}

func dataSourceProjectRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return projectRead(ctx, d, meta, true)
}
