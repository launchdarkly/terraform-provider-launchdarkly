package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceFlagTemplates() *schema.Resource {
	schemaMap := baseFlagTemplatesSchema(true)
	return &schema.Resource{
		ReadContext: dataSourceFlagTemplatesRead,
		Schema:      removeInvalidFieldsForDataSource(schemaMap),

		Description: `Provides a LaunchDarkly flag templates data source.

This data source allows you to retrieve the "Custom" flag template settings for a LaunchDarkly project. LaunchDarkly projects include several built-in flag templates (Release, Kill switch, Experiment, Custom, Migration). This data source reads the Custom template only.`,
	}
}

func dataSourceFlagTemplatesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return flagTemplatesRead(ctx, d, meta, true)
}
