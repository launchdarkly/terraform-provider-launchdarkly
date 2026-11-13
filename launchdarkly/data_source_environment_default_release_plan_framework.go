package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var _ datasource.DataSource = &EnvironmentDefaultReleasePlanDataSource{}

type EnvironmentDefaultReleasePlanDataSource struct {
	client *Client
}

type EnvironmentDefaultReleasePlanDataSourceModel = EnvironmentDefaultReleasePlanResourceModel

func NewEnvironmentDefaultReleasePlanDataSource() datasource.DataSource {
	return &EnvironmentDefaultReleasePlanDataSource{}
}

func (d *EnvironmentDefaultReleasePlanDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_default_release_plan"
}

func (d *EnvironmentDefaultReleasePlanDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly environment default release plan data source.\n\n~> **Beta:** This data source wraps a beta LaunchDarkly API (the `default-release-plan` endpoint, accessed with the `LD-API-Version: beta` header). Beta resources may change or be removed in future versions.\n\nThis data source allows you to look up which release plan is the default for a LaunchDarkly environment. It returns an error when the environment has no default release plan.",
		Attributes: map[string]schema.Attribute{
			ID: schema.StringAttribute{
				Computed:    true,
				Description: "The ID in the format `project_key/environment_key`.",
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The project key of the environment.",
			},
			ENVIRONMENT_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The key of the environment.",
			},
			RELEASE_PLAN_KEY: schema.StringAttribute{
				Computed:    true,
				Description: "The key of the environment's default release plan.",
			},
		},
	}
}

func (d *EnvironmentDefaultReleasePlanDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *EnvironmentDefaultReleasePlanDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data EnvironmentDefaultReleasePlanDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	envKey := data.EnvironmentKey.ValueString()

	plan, _, err := getEnvironmentDefaultReleasePlan(ctx, d.client, projectKey, envKey)
	if err != nil {
		// Surface the raw upstream error so ExpectError regex matches
		// "Error: 404 Not Found:" directly against the summary.
		resp.Diagnostics.AddError(handleLdapiErr(err).Error(), "")
		return
	}
	if plan == nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Environment %q of project %q has no default release plan", envKey, projectKey),
			"Set one with the launchdarkly_environment_default_release_plan resource or in the LaunchDarkly UI.",
		)
		return
	}

	environmentDefaultReleasePlanToModel(projectKey, envKey, plan, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
