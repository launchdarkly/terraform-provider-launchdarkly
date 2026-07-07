package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v23"
)

var _ datasource.DataSource = &FlagImportConfigurationDataSource{}

type FlagImportConfigurationDataSource struct {
	client *Client
}

type FlagImportConfigurationDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectKey     types.String `tfsdk:"project_key"`
	IntegrationKey types.String `tfsdk:"integration_key"`
	IntegrationID  types.String `tfsdk:"integration_id"`
	Name           types.String `tfsdk:"name"`
	Config         types.String `tfsdk:"config"`
	Tags           types.Set    `tfsdk:"tags"`
	Version        types.Int64  `tfsdk:"version"`
}

func NewFlagImportConfigurationDataSource() datasource.DataSource {
	return &FlagImportConfigurationDataSource{}
}

func (d *FlagImportConfigurationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flag_import_configuration"
}

func (d *FlagImportConfigurationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly flag import configuration data source.\n\n~> **Beta:** This data source wraps a beta LaunchDarkly API. Beta resources may change or be removed in future versions.\n\nThis data source allows you to retrieve information about a flag import configuration in your LaunchDarkly project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID in the format `project_key/integration_key/integration_id`.",
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The key of the project the flag import configuration belongs to.",
			},
			INTEGRATION_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The integration key identifying the external feature management system, for example `split`.",
			},
			INTEGRATION_ID: schema.StringAttribute{
				Required:    true,
				Description: "The unique identifier of the flag import configuration.",
			},
			NAME: schema.StringAttribute{
				Computed:    true,
				Description: "A human-friendly name for the flag import configuration.",
			},
			CONFIG: schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "A JSON-encoded object of configuration values for the integration. Secret values may be masked by the API.",
			},
			TAGS: schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Tags associated with the flag import configuration.",
			},
			VERSION: schema.Int64Attribute{
				Computed:    true,
				Description: "The version of the flag import configuration.",
			},
		},
	}
}

func (d *FlagImportConfigurationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *FlagImportConfigurationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data FlagImportConfigurationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	beta, err := newFlagImportConfigurationBetaClient(d.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}

	projectKey := data.ProjectKey.ValueString()
	integrationKey := data.IntegrationKey.ValueString()
	integrationID := data.IntegrationID.ValueString()

	var cfg *ldapi.FlagImportIntegration
	err = beta.withConcurrency(beta.ctx, func() error {
		cfg, _, err = beta.ld.FlagImportConfigurationsBetaApi.GetFlagImportConfiguration(beta.ctx, projectKey, integrationKey, integrationID).Execute()
		return err
	})
	if err != nil {
		// Surface the raw upstream error so ExpectError regex matches
		// "Error: 404 Not Found:" directly against the summary.
		resp.Diagnostics.AddError(handleLdapiErr(err).Error(), "")
		return
	}

	data.ID = types.StringValue(flagImportConfigurationID(projectKey, integrationKey, integrationID))
	data.IntegrationKey = types.StringValue(cfg.GetIntegrationKey())
	data.IntegrationID = types.StringValue(cfg.GetId())
	data.Name = types.StringValue(cfg.GetName())
	data.Version = types.Int64Value(int64(cfg.GetVersion()))

	configJSON, cErr := configJSONFromMap(cfg.GetConfig())
	if cErr != nil {
		resp.Diagnostics.AddError("Failed to read flag import configuration config", cErr.Error())
		return
	}
	data.Config = types.StringValue(configJSON)

	tagsSet, diags := setFromStringSlice(ctx, cfg.GetTags())
	resp.Diagnostics.Append(diags...)
	data.Tags = tagsSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
