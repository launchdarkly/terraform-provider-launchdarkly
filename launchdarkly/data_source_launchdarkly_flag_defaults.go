package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceFlagDefaults() *schema.Resource {
	schemaMap := baseFlagDefaultsSchema(true)
	return &schema.Resource{
		ReadContext: dataSourceFlagDefaultsRead,
		Schema:      removeInvalidFieldsForDataSource(schemaMap),

		Description: `Provides a LaunchDarkly flag defaults data source.

This data source allows you to retrieve the flag default settings for a LaunchDarkly project.`,
	}
}

func dataSourceFlagDefaultsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return flagDefaultsRead(ctx, d, meta, true)
}
