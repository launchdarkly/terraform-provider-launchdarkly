package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SdkKeyDataSource{}

type SdkKeyDataSource struct {
	client *Client
}

type SdkKeyDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectKey     types.String `tfsdk:"project_key"`
	EnvironmentKey types.String `tfsdk:"environment_key"`
	Key            types.String `tfsdk:"key"`
	Kind           types.String `tfsdk:"kind"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Expiry         types.Int64  `tfsdk:"expiry"`
	Value          types.String `tfsdk:"value"`
	IsDefault      types.Bool   `tfsdk:"is_default"`
	Version        types.Int64  `tfsdk:"version"`
}

func NewSdkKeyDataSource() datasource.DataSource {
	return &SdkKeyDataSource{}
}

func (d *SdkKeyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sdk_key"
}

func (d *SdkKeyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Provides a LaunchDarkly SDK key data source.

~> **Beta:** This data source uses a beta API. Beta resources may change or be removed in future versions.

This data source allows you to retrieve information about a specific SDK key in a project environment.`,
		Attributes: map[string]schema.Attribute{
			"id":            schema.StringAttribute{Computed: true, Description: "The unique resource ID in the format `project_key/environment_key/key`."},
			PROJECT_KEY:     schema.StringAttribute{Required: true, Description: "The project key."},
			ENVIRONMENT_KEY: schema.StringAttribute{Required: true, Description: "The environment key."},
			KEY:             schema.StringAttribute{Required: true, Description: "The user-defined identifying key of the SDK key."},
			KIND:            schema.StringAttribute{Computed: true, Description: "The kind of SDK key. Either `sdk` (server-side) or `mobile`."},
			NAME:            schema.StringAttribute{Computed: true, Description: "The human-readable name of the SDK key."},
			DESCRIPTION:     schema.StringAttribute{Computed: true, Description: "The description of the SDK key."},
			EXPIRY:          schema.Int64Attribute{Computed: true, Description: "The expiration date for the SDK key, expressed as a Unix epoch time in milliseconds, if set."},
			VALUE: schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The actual SDK key value. Use this when configuring your SDK.",
			},
			IS_DEFAULT: schema.BoolAttribute{Computed: true, Description: "Whether this SDK key is the system-defined default for the environment."},
			VERSION:    schema.Int64Attribute{Computed: true, Description: "The auto-incremented version number of the SDK key."},
		},
	}
}

func (d *SdkKeyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *SdkKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data SdkKeyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	beta, err := newBetaClient(d.client.apiKey, d.client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		resp.Diagnostics.AddError("Failed to construct beta client", err.Error())
		return
	}

	projectKey := data.ProjectKey.ValueString()
	environmentKey := data.EnvironmentKey.ValueString()
	sdkKeyKey := data.Key.ValueString()

	sdkKey, _, err := getSdkKey(beta, projectKey, environmentKey, sdkKeyKey)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to get SDK key %q in environment %q of project %q: %s", sdkKeyKey, environmentKey, projectKey, handleLdapiErr(err).Error()),
			"",
		)
		return
	}

	data.ID = types.StringValue(sdkKeyID(projectKey, environmentKey, sdkKeyKey))
	data.ProjectKey = types.StringValue(projectKey)
	data.EnvironmentKey = types.StringValue(environmentKey)
	data.Key = types.StringValue(sdkKey.Key)
	data.Kind = types.StringValue(string(sdkKey.Kind))
	data.Name = types.StringValue(sdkKey.Name)
	if sdkKey.Description != nil {
		data.Description = types.StringValue(*sdkKey.Description)
	} else {
		data.Description = types.StringValue("")
	}
	if sdkKey.Expiry != nil {
		data.Expiry = types.Int64Value(*sdkKey.Expiry)
	} else {
		data.Expiry = types.Int64Null()
	}
	data.Value = types.StringValue(sdkKey.Value)
	data.IsDefault = types.BoolValue(sdkKey.IsDefault)
	data.Version = types.Int64Value(int64(sdkKey.Version))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
