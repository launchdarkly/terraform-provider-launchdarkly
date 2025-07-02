package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceView() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceViewRead,

		Description: "Provides a LaunchDarkly view data source.\n\nThis data source allows you to retrieve view information from your LaunchDarkly project.",

		Schema: map[string]*schema.Schema{
			PROJECT_KEY: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The project key.",
			},
			KEY: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The view's unique key.",
			},
			NAME: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The view's name.",
			},
			DESCRIPTION: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The view's description.",
			},
			GENERATE_SDK_KEYS: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether SDK keys are generated for this view.",
			},
			MAINTAINER_ID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The member ID of the maintainer for this view.",
			},
			MAINTAINER_TEAM_KEY: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The team key of the maintainer team for this view.",
			},
			TAGS: tagsSchema(tagsSchemaOptions{isDataSource: true}),
			ARCHIVED: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the view is archived.",
			},
			LINKED_FLAGS: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "A list of feature flag keys that are linked to this view.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceViewRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return viewRead(ctx, d, meta, true)
}
