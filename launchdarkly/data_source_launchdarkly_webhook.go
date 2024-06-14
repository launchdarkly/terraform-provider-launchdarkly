package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceWebhook() *schema.Resource {
	schemaMap := baseWebhookSchema(webhookSchemaOptions{isDataSource: true})
	schemaMap[URL] = &schema.Schema{
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The URL of the remote webhook.",
	}
	schemaMap[ON] = &schema.Schema{
		Type:        schema.TypeBool,
		Computed:    true,
		Description: "Whether the webhook is enabled.",
	}
	schemaMap[ID] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The unique webhook ID.",
	}
	return &schema.Resource{
		ReadContext: dataSourceWebhookRead,
		Schema:      schemaMap,

		Description: `Provides a LaunchDarkly webhook data source.

This data source allows you to retrieve webhook information from your LaunchDarkly organization.`,
	}
}

func dataSourceWebhookRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return webhookRead(ctx, d, meta, true)
}
