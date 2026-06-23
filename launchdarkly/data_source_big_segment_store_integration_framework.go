package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var _ datasource.DataSource = &BigSegmentStoreIntegrationDataSource{}

type BigSegmentStoreIntegrationDataSource struct {
	client *Client
	beta   *Client
}

type BigSegmentStoreIntegrationDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectKey     types.String `tfsdk:"project_key"`
	EnvironmentKey types.String `tfsdk:"environment_key"`
	IntegrationKey types.String `tfsdk:"integration_key"`
	IntegrationID  types.String `tfsdk:"integration_id"`
	Name           types.String `tfsdk:"name"`
	On             types.Bool   `tfsdk:"on"`
	Config         types.String `tfsdk:"config"`
	Tags           types.Set    `tfsdk:"tags"`
	Version        types.Int64  `tfsdk:"version"`
}

func NewBigSegmentStoreIntegrationDataSource() datasource.DataSource {
	return &BigSegmentStoreIntegrationDataSource{}
}

func (d *BigSegmentStoreIntegrationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_big_segment_store_integration"
}

func (d *BigSegmentStoreIntegrationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Provides a LaunchDarkly big segment store (persistent store) integration data source.

~> **Beta:** This data source wraps a beta LaunchDarkly API. Beta resources may change or be removed in future provider versions.

This data source allows you to retrieve a persistent store integration configuration for an environment by its IDs.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID in the format `project_key/environment_key/integration_key/integration_id`.",
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The key of the project the integration belongs to.",
			},
			ENVIRONMENT_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The key of the environment the integration belongs to.",
			},
			INTEGRATION_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The persistent store technology. One of `redis` or `dynamodb`.",
			},
			INTEGRATION_ID: schema.StringAttribute{
				Required:    true,
				Description: "The server-assigned ID of the integration configuration.",
			},
			NAME: schema.StringAttribute{
				Computed:    true,
				Description: "A human-friendly name for the integration configuration.",
			},
			ON: schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the integration is turned on.",
			},
			CONFIG: schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "A JSON string holding the store-specific configuration. Marked sensitive because it carries credentials.",
			},
			TAGS: schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Tags associated with the integration configuration.",
			},
			VERSION: schema.Int64Attribute{
				Computed:    true,
				Description: "The version of the integration configuration.",
			},
		},
	}
}

func (d *BigSegmentStoreIntegrationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
	if d.client == nil {
		return
	}
	beta, err := newBigSegmentStoreIntegrationBetaClient(d.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build LaunchDarkly beta client", err.Error())
		return
	}
	d.beta = beta
}

func (d *BigSegmentStoreIntegrationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data BigSegmentStoreIntegrationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	environmentKey := data.EnvironmentKey.ValueString()
	integrationKey := data.IntegrationKey.ValueString()
	integrationID := data.IntegrationID.ValueString()

	var integration *ldapi.BigSegmentStoreIntegration
	var err error
	err = d.beta.withConcurrency(d.beta.ctx, func() error {
		integration, _, err = d.beta.ld.PersistentStoreIntegrationsBetaApi.GetBigSegmentStoreIntegration(d.beta.ctx, projectKey, environmentKey, integrationKey, integrationID).Execute()
		return err
	})
	if err != nil {
		// Surface the raw upstream error so ExpectError regex matches
		// "Error: 404 Not Found:" directly against the summary.
		resp.Diagnostics.AddError(handleLdapiErr(err).Error(), "")
		return
	}

	data.ID = types.StringValue(bigSegmentStoreIntegrationID(projectKey, environmentKey, integrationKey, integration.GetId()))
	data.ProjectKey = types.StringValue(integration.GetProjectKey())
	data.EnvironmentKey = types.StringValue(integration.GetEnvironmentKey())
	data.IntegrationKey = types.StringValue(integration.GetIntegrationKey())
	data.IntegrationID = types.StringValue(integration.GetId())
	data.Name = types.StringValue(integration.GetName())
	data.On = types.BoolValue(integration.GetOn())
	data.Version = types.Int64Value(int64(integration.GetVersion()))

	configJSON, err := mapToJsonString(integration.GetConfig())
	if err != nil {
		resp.Diagnostics.AddError("Failed to serialise config", err.Error())
		return
	}
	data.Config = types.StringValue(configJSON)

	tagsSet, diags := setFromStringSlice(ctx, integration.Tags)
	resp.Diagnostics.Append(diags...)
	data.Tags = tagsSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
