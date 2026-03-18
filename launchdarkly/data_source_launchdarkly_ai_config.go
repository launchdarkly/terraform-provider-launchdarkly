package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAIConfig() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceAIConfigRead,

		Description: "Provides a LaunchDarkly AI config data source.\n\nThis data source allows you to retrieve AI configuration information from your LaunchDarkly project.",

		Schema: baseAIConfigSchema(true),
	}
}

func dataSourceAIConfigRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return aiConfigRead(ctx, d, meta, true)
}
