package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceWebhook() *schema.Resource {
	schemaMap := baseWebhookSchema(webhookSchemaOptions{isDataSource: true})
	schemaMap[URL] = &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	}
	schemaMap[ON] = &schema.Schema{
		Type:     schema.TypeBool,
		Computed: true,
	}
	schemaMap[ID] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The ID of the webhook",
	}
	return &schema.Resource{
		ReadContext: dataSourceWebhookRead,
		Schema:      schemaMap,
	}
}

func dataSourceWebhookRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return webhookRead(ctx, d, meta, true)
}
