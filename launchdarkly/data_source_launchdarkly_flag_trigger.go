package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceFlagTrigger() *schema.Resource {
	schemaMap := baseFlagTriggerSchema(true)
	schemaMap[ID] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The Terraform trigger ID. The unique trigger ID can be found in your saved trigger URL:\n\n```\nhttps://app.launchdarkly.com/webhook/triggers/THIS_IS_YOUR_TRIGGER_ID/aff25a53-17d9-4112-a9b8-12718d1a2e79\n```\n\nPlease note that if you did not save this upon creation of the resource, you will have to reset it to get a new value, which can cause breaking changes.",
	}
	return &schema.Resource{
		ReadContext: dataSourceFlagTriggerRead,
		Schema:      removeInvalidFieldsForDataSource(schemaMap),

		Description: `Provides a LaunchDarkly flag trigger data source.

-> **Note:** Flag triggers are available to customers on an Enterprise LaunchDarkly plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).

This data source allows you to retrieve information about flag triggers from your LaunchDarkly organization.`,
	}
}

func dataSourceFlagTriggerRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return flagTriggerRead(ctx, d, metaRaw, true)
}
