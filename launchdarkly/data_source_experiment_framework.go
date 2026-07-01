package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var _ datasource.DataSource = &ExperimentDataSource{}

type ExperimentDataSource struct {
	client *Client
}

type ExperimentDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectKey     types.String `tfsdk:"project_key"`
	EnvironmentKey types.String `tfsdk:"environment_key"`
	Key            types.String `tfsdk:"key"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	MaintainerID   types.String `tfsdk:"maintainer_id"`
	HoldoutID      types.String `tfsdk:"holdout_id"`
	Tags           types.Set    `tfsdk:"tags"`
	Archived       types.Bool   `tfsdk:"archived"`
}

func NewExperimentDataSource() datasource.DataSource {
	return &ExperimentDataSource{}
}

func (d *ExperimentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_experiment"
}

func (d *ExperimentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly experiment data source.\n\nThis data source allows you to retrieve information about an experiment in an environment.",
		Attributes: map[string]schema.Attribute{
			ID: schema.StringAttribute{
				Computed:    true,
				Description: "The ID of this resource, in the format `project_key/environment_key/experiment_key`.",
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The project key.",
			},
			ENVIRONMENT_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The environment key.",
			},
			KEY: schema.StringAttribute{
				Required:    true,
				Description: "The unique key that references the experiment.",
			},
			NAME: schema.StringAttribute{
				Computed:    true,
				Description: "The human-friendly name of the experiment.",
			},
			DESCRIPTION: schema.StringAttribute{
				Computed:    true,
				Description: "A description of the experiment.",
			},
			MAINTAINER_ID: schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the member who maintains the experiment.",
			},
			HOLDOUT_ID: schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the holdout associated with this experiment.",
			},
			TAGS: schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Tags associated with the experiment.",
			},
			ARCHIVED: schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the experiment is archived.",
			},
		},
	}
}

func (d *ExperimentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *ExperimentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data ExperimentDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	environmentKey := data.EnvironmentKey.ValueString()
	key := data.Key.ValueString()

	var experiment *ldapi.Experiment
	err := d.client.withConcurrency(d.client.ctx, func() error {
		var e error
		experiment, _, e = d.client.ld.ExperimentsApi.GetExperiment(d.client.ctx, projectKey, environmentKey, key).Execute()
		return e
	})
	if err != nil {
		resp.Diagnostics.AddError(handleLdapiErr(err).Error(), "")
		return
	}

	data.ID = types.StringValue(experimentID(projectKey, environmentKey, key))
	data.Name = types.StringValue(experiment.Name)
	data.Description = stringValueFromPointer(experiment.Description)
	data.MaintainerID = types.StringValue(experiment.MaintainerId)
	data.HoldoutID = stringValueFromPointer(experiment.HoldoutId)
	data.Archived = types.BoolValue(experiment.ArchivedDate != nil)

	tags, diags := setFromStringSlice(ctx, experiment.Tags)
	resp.Diagnostics.Append(diags...)
	data.Tags = tags

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
