package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceRelayProxyConfig() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceRelayProxyRead,

		Schema: map[string]*schema.Schema{
			ID: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The Relay Proxy configuration's unique 24 character ID",
			},
			NAME: {
				Type:        schema.TypeString,
				Description: "A human-friendly name for the Relay Proxy configuration",
				Computed:    true,
			},
			POLICY: policyStatementsSchema(policyStatementSchemaOptions{required: false}),
			DISPLAY_KEY: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The last four characters of the full_key.",
			},
		},
	}
}

func dataSourceRelayProxyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Get(ID).(string)
	d.SetId(id)
	return relayProxyConfigRead(ctx, d, m, true)
}
