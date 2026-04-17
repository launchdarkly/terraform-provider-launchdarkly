package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAIConfigVariation() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceAIConfigVariationRead,

		Description: "Provides a LaunchDarkly AI Config variation data source.\n\nThis data source allows you to retrieve AI Config variation information from your LaunchDarkly project.",

		Schema: aiConfigVariationSchema(true),
	}
}

func dataSourceAIConfigVariationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return aiConfigVariationRead(ctx, d, meta, true)
}
