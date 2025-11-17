package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAIConfig() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceAIConfigRead,
		Schema: map[string]*schema.Schema{
			PROJECT_KEY: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The project key.",
			},
			KEY: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The unique key of the AI Config.",
			},
			NAME: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The human-readable name of the AI Config.",
			},
			DESCRIPTION: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The description of the AI Config.",
			},
			TAGS: {
				Type:        schema.TypeSet,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Tags associated with the AI Config.",
			},
			MAINTAINER_ID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the member who maintains this AI Config.",
			},
			MAINTAINER_TEAM_KEY: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The key of the team that maintains this AI Config.",
			},
			VERSION: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The version of the AI Config.",
			},
		},
		Description: `Provides a LaunchDarkly AI Config data source.

This data source allows you to retrieve AI Config information from your LaunchDarkly project.

-> **Note:** AI Configs are currently in beta.`,
	}
}

func dataSourceAIConfigRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)
	d.SetId(projectKey + "/" + key)
	return resourceAIConfigRead(ctx, d, metaRaw)
}
