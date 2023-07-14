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
			access_token: schema.StringAttribute{
				Optional:    true,
				Description: sdkProviderSchema[access_token].Description,
			},
			oauth_token: schema.StringAttribute{
				Optional:    true,
				Description: sdkProviderSchema[oauth_token].Description,
			},
			api_host: schema.StringAttribute{
				Optional:    true,
				Description: sdkProviderSchema[api_host].Description,
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

	if strings.HasPrefix(host, "http") {
		u, _ := url.Parse(host)
		host = u.Host
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

	if oauthToken == "" && accessToken == "" {
		resp.Diagnostics.AddError("Missing authentication token", fmt.Sprintf("Either the %q or %q must be specified.", access_token, oauth_token))
		return
	}

	if oauthToken != "" {
		client, err := newClient(oauthToken, host, true)
		if err != nil {
			resp.Diagnostics.AddError("Unable to create LaunchDarkly client", err.Error())
			return
		}
		resp.ResourceData = client
		return
	}

	client, err := newClient(accessToken, host, false)
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
	return nil
}

func NewPluginProvider(version string) func() provider.Provider {
	return func() provider.Provider {
		return &launchdarklyProvider{
			version: version,
		}
	}
}
