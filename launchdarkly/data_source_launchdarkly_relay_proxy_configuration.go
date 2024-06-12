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
				Description: "The Relay Proxy configuration's unique 24 character ID. The unique relay proxy ID can be found in the relay proxy edit page URL, which you can locate by clicking the three dot menu on your relay proxy item in the UI and selecting \"Edit configuration\":\n\n```\nhttps://app.launchdarkly.com/settings/relay/THIS_IS_YOUR_RELAY_PROXY_ID/edit\n```",
			},
			NAME: {
				Type:        schema.TypeString,
				Description: "The human-readable name for your Relay Proxy configuration.",
				Computed:    true,
			},
			POLICY: policyStatementsSchema(policyStatementSchemaOptions{
				required:    false,
				computed:    true,
				description: `The Relay Proxy configuration's rule policy block. This determines what content the Relay Proxy receives. To learn more, read [Understanding policies](https://docs.launchdarkly.com/home/members/role-policies#understanding-policies).`,
			}),
			DISPLAY_KEY: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The last 4 characters of the Relay Proxy configuration's unique key.",
			},
		},

		Description: "Provides a LaunchDarkly Relay Proxy configuration data source for use with the Relay Proxy's [automatic configuration feature](https://docs.launchdarkly.com/home/relay-proxy/automatic-configuration).\n\n-> **Note:** Relay Proxy automatic configuration is available to customers on an Enterprise LaunchDarkly plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).\n\nThis data source allows you to retrieve Relay Proxy configuration information from your LaunchDarkly organization.\n\n-> **Note:** It is not possible for this data source to retrieve your Relay Proxy configuration's unique key. This is because the unique key is only exposed upon creation. If you need to reference the Relay Proxy configuration's unique key in your terraform config, use the `launchdarkly_relay_proxy_configuration` resource instead.",
	}
}

func dataSourceRelayProxyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Get(ID).(string)
	d.SetId(id)
	return relayProxyConfigRead(ctx, d, m, true)
}
