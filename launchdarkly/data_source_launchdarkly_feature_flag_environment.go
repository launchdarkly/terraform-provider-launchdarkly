package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceFeatureFlagEnvironment() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceFeatureFlagEnvironmentRead,
		Schema:      baseFeatureFlagEnvironmentSchema(true),
	}
}

func dataSourceFeatureFlagEnvironmentRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return featureFlagEnvironmentRead(ctx, d, meta, true)
}
