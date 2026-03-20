package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAITool() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceAIToolRead,

		Description: "Provides a LaunchDarkly AI tool data source.\n\nThis data source allows you to retrieve AI tool information from your LaunchDarkly project.",

		Schema: baseAIToolSchema(true),
	}
}

func dataSourceAIToolRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return aiToolRead(ctx, d, meta, true)
}
