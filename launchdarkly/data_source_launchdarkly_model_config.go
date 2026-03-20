package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceModelConfig() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceModelConfigRead,

		Description: "Provides a LaunchDarkly model config data source.\n\nThis data source allows you to retrieve AI model configuration information from your LaunchDarkly project.",

		Schema: baseModelConfigSchema(true),
	}
}

func dataSourceModelConfigRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return modelConfigRead(ctx, d, meta, true)
}
