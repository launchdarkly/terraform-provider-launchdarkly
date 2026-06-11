package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var _ datasource.DataSource = &MetricGroupDataSource{}

type MetricGroupDataSource struct {
	client *Client
}

type MetricGroupDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	ProjectKey   types.String `tfsdk:"project_key"`
	Key          types.String `tfsdk:"key"`
	Name         types.String `tfsdk:"name"`
	Kind         types.String `tfsdk:"kind"`
	Description  types.String `tfsdk:"description"`
	MaintainerID types.String `tfsdk:"maintainer_id"`
	Tags         types.Set    `tfsdk:"tags"`
	Metrics      types.List   `tfsdk:"metrics"`
	Version      types.Int64  `tfsdk:"version"`
}

func NewMetricGroupDataSource() datasource.DataSource {
	return &MetricGroupDataSource{}
}

func (d *MetricGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_metric_group"
}

func (d *MetricGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly metric group data source.\n\n~> **Beta:** This data source uses a beta LaunchDarkly API. Beta resources may change or be removed in future versions.\n\nThis data source allows you to retrieve metric group information from your LaunchDarkly project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID in the format `project_key/key`.",
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The metric group's project key.",
			},
			KEY: schema.StringAttribute{
				Required:    true,
				Description: "The unique key that references the metric group.",
			},
			NAME: schema.StringAttribute{
				Computed:    true,
				Description: "The human-friendly name for the metric group.",
			},
			KIND: schema.StringAttribute{
				Computed:    true,
				Description: "The type of the metric group. One of `funnel` or `standard`.",
			},
			DESCRIPTION: schema.StringAttribute{
				Computed:    true,
				Description: "A description of the metric group's purpose.",
			},
			MAINTAINER_ID: schema.StringAttribute{
				Computed:    true,
				Description: "The LaunchDarkly member ID of the maintainer.",
			},
			TAGS: schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Tags associated with the metric group.",
			},
			VERSION: schema.Int64Attribute{
				Computed:    true,
				Description: "The version of the metric group.",
			},
			METRICS: schema.ListNestedAttribute{
				Computed:    true,
				Description: "An ordered list of the metrics in this metric group.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						KEY: schema.StringAttribute{
							Computed:    true,
							Description: "The key of the metric.",
						},
						NAME_IN_GROUP: schema.StringAttribute{
							Computed:    true,
							Description: "The name of the metric when used within this metric group.",
						},
					},
				},
			},
		},
	}
}

func (d *MetricGroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *MetricGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data MetricGroupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	beta, err := newMetricGroupBetaClient(d.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}

	projectKey := data.ProjectKey.ValueString()
	key := data.Key.ValueString()

	var group *ldapi.MetricGroupRep
	err = beta.withConcurrency(beta.ctx, func() error {
		group, _, err = beta.ld.MetricsBetaApi.GetMetricGroup(beta.ctx, projectKey, key).Execute()
		return err
	})
	if err != nil {
		// Surface the raw upstream error so ExpectError regex matches
		// "Error: 404 Not Found:" directly against the summary.
		resp.Diagnostics.AddError(handleLdapiErr(err).Error(), "")
		return
	}

	data.ID = types.StringValue(projectKey + "/" + key)
	data.Key = types.StringValue(group.Key)
	data.Name = types.StringValue(group.Name)
	data.Kind = types.StringValue(group.Kind)
	if group.Description != nil {
		data.Description = types.StringValue(*group.Description)
	} else {
		data.Description = types.StringValue("")
	}
	data.Version = types.Int64Value(int64(group.Version))
	if group.Maintainer.Member != nil && group.Maintainer.Member.GetId() != "" {
		data.MaintainerID = types.StringValue(group.Maintainer.Member.GetId())
	} else {
		data.MaintainerID = types.StringValue("")
	}

	tagsSet, diags := setFromStringSlice(ctx, group.Tags)
	resp.Diagnostics.Append(diags...)
	data.Tags = tagsSet

	metricsList, err := metricGroupMetricsToList(group.Metrics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read metric group metrics", err.Error())
		return
	}
	data.Metrics = metricsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
