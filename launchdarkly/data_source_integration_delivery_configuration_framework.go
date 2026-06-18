package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var _ datasource.DataSource = &IntegrationDeliveryConfigurationDataSource{}

type IntegrationDeliveryConfigurationDataSource struct {
	client *Client
}

type IntegrationDeliveryConfigurationDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectKey     types.String `tfsdk:"project_key"`
	EnvKey         types.String `tfsdk:"env_key"`
	IntegrationKey types.String `tfsdk:"integration_key"`
	ConfigID       types.String `tfsdk:"config_id"`
	Name           types.String `tfsdk:"name"`
	Config         types.String `tfsdk:"config"`
	On             types.Bool   `tfsdk:"on"`
	Tags           types.Set    `tfsdk:"tags"`
	Version        types.Int64  `tfsdk:"version"`
}

func NewIntegrationDeliveryConfigurationDataSource() datasource.DataSource {
	return &IntegrationDeliveryConfigurationDataSource{}
}

func (d *IntegrationDeliveryConfigurationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_delivery_configuration"
}

func (d *IntegrationDeliveryConfigurationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly integration delivery configuration data source.\n\n~> **Beta:** This data source wraps a beta LaunchDarkly API. Beta resources may change or be removed in future versions.\n\nThis data source allows you to retrieve an integration delivery configuration from your LaunchDarkly project environment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of this resource in the format `project_key/env_key/integration_key/config_id`.",
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The project key.",
			},
			ENV_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The environment key.",
			},
			INTEGRATION_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The integration key identifying the persistent feature store integration (for example `fastly`, `cloudflare`, or `vercel`).",
			},
			CONFIG_ID: schema.StringAttribute{
				Required:    true,
				Description: "The unique server-assigned ID of the delivery configuration.",
			},
			NAME: schema.StringAttribute{
				Computed:    true,
				Description: "A human-friendly name for the delivery configuration.",
			},
			CONFIG: schema.StringAttribute{
				Computed:    true,
				Description: "A JSON string representing the integration-specific configuration.",
			},
			ON: schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the delivery configuration is turned on.",
			},
			TAGS: schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Tags associated with the delivery configuration.",
			},
			VERSION: schema.Int64Attribute{
				Computed:    true,
				Description: "The version of the delivery configuration.",
			},
		},
	}
}

func (d *IntegrationDeliveryConfigurationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *IntegrationDeliveryConfigurationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data IntegrationDeliveryConfigurationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	beta, err := newIntegrationDeliveryConfigurationBetaClient(d.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}

	projectKey := data.ProjectKey.ValueString()
	envKey := data.EnvKey.ValueString()
	integrationKey := data.IntegrationKey.ValueString()
	configID := data.ConfigID.ValueString()

	var cfg *ldapi.IntegrationDeliveryConfiguration
	err = beta.withConcurrency(beta.ctx, func() error {
		cfg, _, err = beta.ld.IntegrationDeliveryConfigurationsBetaApi.
			GetIntegrationDeliveryConfigurationById(beta.ctx, projectKey, envKey, integrationKey, configID).
			Execute()
		return err
	})
	if err != nil {
		// Surface the raw upstream error so ExpectError regex matches against
		// the summary directly.
		resp.Diagnostics.AddError(handleLdapiErr(err).Error(), "")
		return
	}

	data.ID = types.StringValue(integrationDeliveryConfigurationID(projectKey, envKey, integrationKey, cfg.GetId()))
	data.ProjectKey = types.StringValue(cfg.GetProjectKey())
	data.EnvKey = types.StringValue(cfg.GetEnvironmentKey())
	data.IntegrationKey = types.StringValue(cfg.GetIntegrationKey())
	data.ConfigID = types.StringValue(cfg.GetId())
	data.Name = types.StringValue(cfg.GetName())
	data.On = types.BoolValue(cfg.GetOn())
	data.Version = types.Int64Value(int64(cfg.GetVersion()))

	configJSON, err := mapToJsonString(cfg.GetConfig())
	if err != nil {
		resp.Diagnostics.AddError("Failed to serialise config", err.Error())
		return
	}
	data.Config = types.StringValue(configJSON)

	tagsSet, diags := setFromStringSlice(ctx, cfg.GetTags())
	resp.Diagnostics.Append(diags...)
	data.Tags = tagsSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
