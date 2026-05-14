package launchdarkly

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ provider.Provider = &launchdarklyProvider{}
)

type launchdarklyProvider struct {
	version string
}

type launchdarklyProviderModel struct {
	AccessToken types.String `tfsdk:"access_token"`
	OAuthToken  types.String `tfsdk:"oauth_token"`
	Host        types.String `tfsdk:"api_host"`
	HttpTimeout types.Int64  `tfsdk:"http_timeout"`
}

// Metadata returns the provider type name.
func (p *launchdarklyProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "launchdarkly"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *launchdarklyProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			ACCESS_TOKEN: schema.StringAttribute{
				Optional:    true,
				Description: "The [personal access token](https://docs.launchdarkly.com/home/account-security/api-access-tokens#personal-tokens) or [service token](https://docs.launchdarkly.com/home/account-security/api-access-tokens#service-tokens) used to authenticate with LaunchDarkly. You can also set this with the `LAUNCHDARKLY_ACCESS_TOKEN` environment variable. You must provide either `access_token` or `oauth_token`.",
			},
			OAUTH_TOKEN: schema.StringAttribute{
				Optional:    true,
				Description: "An OAuth V2 token you use to authenticate with LaunchDarkly. You can also set this with the `LAUNCHDARKLY_OAUTH_TOKEN` environment variable. You must provide either `access_token` or `oauth_token`.",
			},
			API_HOST: schema.StringAttribute{
				Optional:    true,
				Description: "The LaunchDarkly host address. If this argument is not specified, the default host address is `https://app.launchdarkly.com`",
			},
			HTTP_TIMEOUT: schema.Int64Attribute{
				Optional:    true,
				Description: "The HTTP timeout (in seconds) when making API calls to LaunchDarkly. Defaults to 20 seconds.",
			},
		},
	}
}

// Configure prepares a LaunchDarkly API client for data sources and resources.
func (p *launchdarklyProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// check environment variables first
	accessToken := os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN)
	oauthToken := os.Getenv(LAUNCHDARKLY_OAUTH_TOKEN)
	host := os.Getenv(LAUNCHDARKLY_API_HOST)
	if host == "" {
		host = DEFAULT_LAUNCHDARKLY_HOST
	}

	var data launchdarklyProviderModel

	// Read configuration into data model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if data.AccessToken.ValueString() != "" {
		accessToken = data.AccessToken.ValueString()
	}
	if data.OAuthToken.ValueString() != "" {
		oauthToken = data.OAuthToken.ValueString()
	}
	if data.Host.ValueString() != "" {
		host = data.Host.ValueString()
	}

	if strings.HasPrefix(host, "http") {
		u, _ := url.Parse(host)
		host = u.Host
	}

	httpTimeoutSeconds := int(data.HttpTimeout.ValueInt64())
	if httpTimeoutSeconds == 0 {
		httpTimeoutSeconds = DEFAULT_HTTP_TIMEOUT_S
	}

	if oauthToken == "" && accessToken == "" {
		resp.Diagnostics.AddError("Missing authentication token", fmt.Sprintf("Either the %q or %q must be specified.", ACCESS_TOKEN, OAUTH_TOKEN))
		return
	}

	if oauthToken != "" {
		client, err := newClient(oauthToken, host, true, httpTimeoutSeconds, DEFAULT_MAX_CONCURRENCY)
		if err != nil {
			resp.Diagnostics.AddError("Unable to create LaunchDarkly client", err.Error())
			return
		}
		resp.ResourceData = client
		resp.DataSourceData = client
		return
	}

	client, err := newClient(accessToken, host, false, httpTimeoutSeconds, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create LaunchDarkly client", err.Error())
		return
	}
	resp.ResourceData = client
	resp.DataSourceData = client
}

// DataSources defines the data sources implemented in the provider.
func (p *launchdarklyProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAIConfigDataSource,
		NewAIConfigVariationDataSource,
		NewAIToolDataSource,
		NewAuditLogSubscriptionDataSource,
		NewEnvironmentDataSource,
		NewFeatureFlagDataSource,
		NewFeatureFlagEnvironmentDataSource,
		NewFlagTemplatesDataSource,
		NewFlagTriggerDataSource,
		NewMetricDataSource,
		NewModelConfigDataSource,
		NewProjectDataSource,
		NewRelayProxyConfigurationDataSource,
		NewSegmentDataSource,
		NewTeamDataSource,
		NewTeamMemberDataSource,
		NewTeamMembersDataSource,
		NewViewDataSource,
		NewWebhookDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *launchdarklyProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAccessTokenResource,
		NewAIConfigResource,
		NewAIConfigVariationResource,
		NewAIToolResource,
		NewAuditLogSubscriptionResource,
		NewCustomRoleResource,
		NewDestinationResource,
		NewEnvironmentResource,
		NewFlagTemplatesResource,
		NewIPAllowlistConfigResource,
		NewIPAllowlistEntryResource,
		NewMetricResource,
		NewRelayProxyConfigResource,
		NewTeamMemberResource,
		NewTeamResource,
		NewViewFilterLinksResource,
		NewViewLinksResource,
		NewViewResource,
		NewWebhookResource,
		NewFlagTriggerResource,
		NewModelConfigResource,
		NewTeamRoleMappingResource,
		NewProjectResource,
		NewSegmentResource,
		NewFeatureFlagResource,
		NewFeatureFlagEnvironmentResource,
	}
}

func NewPluginProvider(version string) func() provider.Provider {
	return func() provider.Provider {
		return &launchdarklyProvider{
			version: version,
		}
	}
}
