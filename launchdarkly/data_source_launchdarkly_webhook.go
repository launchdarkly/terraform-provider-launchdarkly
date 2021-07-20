package launchdarkly

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceWebhook() *schema.Resource {
	schemaMap := baseWebhookSchema()
	schemaMap[URL] = &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	}
	schemaMap[ENABLED] = &schema.Schema{
		Type:       schema.TypeBool,
		Computed:   true,
		Deprecated: "'enabled' is deprecated in favor of 'on'",
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
		Read:   dataSourceWebhookRead,
		Schema: schemaMap,
	}
}

func dataSourceWebhookRead(d *schema.ResourceData, meta interface{}) error {
	return webhookRead(d, meta, true)
}
