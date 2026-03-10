package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAiConfig() *schema.Resource {
	schemaMap := baseAiConfigSchema(true)
	return &schema.Resource{
		ReadContext: dataSourceAiConfigRead,
		Schema:      schemaMap,
		Description: `Provides a LaunchDarkly AI Config data source.

This data source allows you to retrieve AI Config information from your LaunchDarkly organization.`,
	}
}

func dataSourceAiConfigRead(ctx context.Context, d *schema.ResourceData, raw interface{}) diag.Diagnostics {
	return aiConfigRead(ctx, d, raw, true)
}
