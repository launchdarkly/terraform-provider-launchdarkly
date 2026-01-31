package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAIModelConfig() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceAIModelConfigRead,
		Schema:      baseAIModelConfigSchema(true),

		Description: `Provides a LaunchDarkly AI model config data source.

This data source allows you to retrieve AI model config information from your LaunchDarkly organization.`,
	}
}

func dataSourceAIModelConfigRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return aiModelConfigRead(ctx, d, metaRaw, true)
}
