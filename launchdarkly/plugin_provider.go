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
	AccessToken           types.String `tfsdk:"access_token"`
	OAuthToken            types.String `tfsdk:"oauth_token"`
	Host                  types.String `tfsdk:"api_host"`
	HttpTimeout           types.Int64  `tfsdk:"http_timeout"`
	MaxConcurrentRequests types.Int64  `tfsdk:"max_concurrent_requests"`
}

// Metadata returns the provider type name.
func (p *launchdarklyProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "launchdarkly"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *launchdarklyProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	sdkProviderSchema := providerSchema()
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			ACCESS_TOKEN: schema.StringAttribute{
				Optional:    true,
				Description: sdkProviderSchema[ACCESS_TOKEN].Description,
			},
			OAUTH_TOKEN: schema.StringAttribute{
				Optional:    true,
				Description: sdkProviderSchema[OAUTH_TOKEN].Description,
			},
			API_HOST: schema.StringAttribute{
				Optional:    true,
				Description: sdkProviderSchema[API_HOST].Description,
			},
			HTTP_TIMEOUT: schema.Int64Attribute{
				Optional:    true,
				Description: sdkProviderSchema[HTTP_TIMEOUT].Description,
			},
			MAX_CONCURRENT_REQUESTS: schema.Int64Attribute{
				Optional:    true,
				Description: sdkProviderSchema[MAX_CONCURRENT_REQUESTS].Description,
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

	maxConcurrent := int(data.MaxConcurrentRequests.ValueInt64())
	if maxConcurrent == 0 {
		maxConcurrent = DEFAULT_MAX_CONCURRENCY
	}
	if maxConcurrent < 0 {
		resp.Diagnostics.AddError("Invalid parallelism", "parallelism must be a positive integer")
	}

	if oauthToken != "" {
		client, err := newClient(oauthToken, host, true, httpTimeoutSeconds, maxConcurrent)
		if err != nil {
			resp.Diagnostics.AddError("Unable to create LaunchDarkly client", err.Error())
			return
		}
		resp.ResourceData = client
		return
	}

	client, err := newClient(accessToken, host, false, httpTimeoutSeconds, maxConcurrent)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create LaunchDarkly client", err.Error())
		return
	}
	resp.ResourceData = client
}

// DataSources defines the data sources implemented in the provider.
func (p *launchdarklyProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

// Resources defines the resources implemented in the provider.
func (p *launchdarklyProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewTeamRoleMappingResource,
	}
}

func NewPluginProvider(version string) func() provider.Provider {
	return func() provider.Provider {
		return &launchdarklyProvider{
			version: version,
		}
	}
}
