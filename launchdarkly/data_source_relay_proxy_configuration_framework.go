package launchdarkly

// data_source_relay_proxy_configuration_framework.go is the
// terraform-plugin-framework implementation of
// launchdarkly_relay_proxy_configuration. The SDKv2 file lives at
// data_source_launchdarkly_relay_proxy_configuration.go (removed in
// this commit).

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var _ datasource.DataSource = &RelayProxyConfigurationDataSource{}

type RelayProxyConfigurationDataSource struct {
	client *Client
}

type RelayProxyConfigurationDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Policy     types.List   `tfsdk:"policy"`
	DisplayKey types.String `tfsdk:"display_key"`
}

func NewRelayProxyConfigurationDataSource() datasource.DataSource {
	return &RelayProxyConfigurationDataSource{}
}

func (d *RelayProxyConfigurationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_relay_proxy_configuration"
}

func (d *RelayProxyConfigurationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly Relay Proxy configuration data source for use with the Relay Proxy's [automatic configuration feature](https://docs.launchdarkly.com/home/relay-proxy/automatic-configuration).\n\n-> **Note:** Relay Proxy automatic configuration is available to customers on an Enterprise LaunchDarkly plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).\n\nThis data source allows you to retrieve Relay Proxy configuration information from your LaunchDarkly organization.\n\n-> **Note:** It is not possible for this data source to retrieve your Relay Proxy configuration's unique key. This is because the unique key is only exposed upon creation. If you need to reference the Relay Proxy configuration's unique key in your terraform config, use the `launchdarkly_relay_proxy_configuration` resource instead.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The Relay Proxy configuration's unique 24 character ID. The unique relay proxy ID can be found in the relay proxy edit page URL, which you can locate by clicking the three dot menu on your relay proxy item in the UI and selecting \"Edit configuration\":\n\n```\nhttps://app.launchdarkly.com/settings/relay/THIS_IS_YOUR_RELAY_PROXY_ID/edit\n```",
			},
			NAME: schema.StringAttribute{
				Computed:    true,
				Description: "The human-readable name for your Relay Proxy configuration.",
			},
			DISPLAY_KEY: schema.StringAttribute{
				Computed:    true,
				Description: "The last 4 characters of the Relay Proxy configuration's unique key.",
			},
			POLICY: frameworkPolicyStatementsDataSourceAttribute("The Relay Proxy configuration's rule policy block. This determines what content the Relay Proxy receives. To learn more, read [Understanding policies](https://docs.launchdarkly.com/home/members/role-policies#understanding-policies)."),
		},
	}
}

func (d *RelayProxyConfigurationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *RelayProxyConfigurationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data RelayProxyConfigurationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	var proxyConfig *ldapi.RelayAutoConfigRep
	var res *http.Response
	var err error
	err = d.client.withConcurrency(d.client.ctx, func() error {
		proxyConfig, res, err = d.client.ld.RelayProxyConfigurationsApi.GetRelayProxyConfig(d.client.ctx, id).Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			resp.Diagnostics.AddError("Relay Proxy configuration not found", fmt.Sprintf("Relay Proxy configuration with id %q not found.", id))
			return
		}
		addLdapiError(&resp.Diagnostics, "Failed to get Relay Proxy configuration", err)
		return
	}

	data.ID = types.StringValue(proxyConfig.Id)
	data.Name = types.StringValue(proxyConfig.Name)
	data.DisplayKey = types.StringValue(proxyConfig.DisplayKey)

	policy, diags := frameworkPolicyStatementsValue(ctx, proxyConfig.Policy)
	resp.Diagnostics.Append(diags...)
	data.Policy = policy

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
