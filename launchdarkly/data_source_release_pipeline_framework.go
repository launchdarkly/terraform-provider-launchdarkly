package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var _ datasource.DataSource = &ReleasePipelineDataSource{}

type ReleasePipelineDataSource struct {
	client *Client
}

type ReleasePipelineDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectKey       types.String `tfsdk:"project_key"`
	Key              types.String `tfsdk:"key"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	Tags             types.Set    `tfsdk:"tags"`
	IsProjectDefault types.Bool   `tfsdk:"is_project_default"`
	CreatedAt        types.String `tfsdk:"created_at"`
	Version          types.Int64  `tfsdk:"version"`
	Phases           types.List   `tfsdk:"phases"`
}

func NewReleasePipelineDataSource() datasource.DataSource {
	return &ReleasePipelineDataSource{}
}

func (d *ReleasePipelineDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_release_pipeline"
}

func (d *ReleasePipelineDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly release pipeline data source.\n\n~> **Beta:** This data source wraps a beta LaunchDarkly API. Beta resources may change or be removed in future versions.\n\nThis data source allows you to retrieve release pipeline information from a LaunchDarkly project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID in the format `project_key/key`.",
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The release pipeline's project key.",
			},
			KEY: schema.StringAttribute{
				Required:    true,
				Description: "The unique key that references the release pipeline.",
			},
			NAME: schema.StringAttribute{
				Computed:    true,
				Description: "A human-friendly name for the release pipeline.",
			},
			DESCRIPTION: schema.StringAttribute{
				Computed:    true,
				Description: "A description of the release pipeline's purpose.",
			},
			TAGS: schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Tags associated with the release pipeline.",
			},
			IS_PROJECT_DEFAULT: schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this release pipeline is the default pipeline for its project.",
			},
			CREATED_AT: schema.StringAttribute{
				Computed:    true,
				Description: "The release pipeline's creation date represented as a UNIX-style timestamp, in milliseconds since the epoch.",
			},
			VERSION: schema.Int64Attribute{
				Computed:    true,
				Description: "The version of the release pipeline.",
			},
			PHASES: schema.ListNestedAttribute{
				Computed:    true,
				Description: "An ordered list of the release pipeline's phases.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						NAME: schema.StringAttribute{
							Computed:    true,
							Description: "The release phase name.",
						},
						AUDIENCES: schema.ListNestedAttribute{
							Computed:    true,
							Description: "An ordered list of the audiences for this phase.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									ENVIRONMENT_KEY: schema.StringAttribute{
										Computed:    true,
										Description: "The key of the LaunchDarkly environment this audience targets.",
									},
									NAME: schema.StringAttribute{
										Computed:    true,
										Description: "The audience name.",
									},
									SEGMENT_KEYS: schema.SetAttribute{
										Computed:    true,
										ElementType: types.StringType,
										Description: "Segment keys targeted by this audience.",
									},
									CONFIGURATION: schema.SingleNestedAttribute{
										Computed:    true,
										Description: "Release strategy and approval configuration for this audience.",
										Attributes: map[string]schema.Attribute{
											RELEASE_STRATEGY: schema.StringAttribute{
												Computed:    true,
												Description: "The release strategy for this audience.",
											},
											REQUIRE_APPROVAL: schema.BoolAttribute{
												Computed:    true,
												Description: "Whether this audience requires approval before changes are rolled out.",
											},
											NOTIFY_MEMBER_IDS: schema.SetAttribute{
												Computed:    true,
												ElementType: types.StringType,
												Description: "Member IDs notified to review the approval request.",
											},
											NOTIFY_TEAM_KEYS: schema.SetAttribute{
												Computed:    true,
												ElementType: types.StringType,
												Description: "Team keys whose members are notified to review the approval request.",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *ReleasePipelineDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *ReleasePipelineDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data ReleasePipelineDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	beta, err := newReleasePipelineBetaClient(d.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}

	projectKey := data.ProjectKey.ValueString()
	key := data.Key.ValueString()

	var pipeline *ldapi.ReleasePipeline
	err = beta.withConcurrency(beta.ctx, func() error {
		pipeline, _, err = beta.ld.ReleasePipelinesBetaApi.GetReleasePipelineByKey(beta.ctx, projectKey, key).Execute()
		return err
	})
	if err != nil {
		// Surface the raw upstream error so ExpectError regex matches
		// "Error: 404 Not Found:" directly against the summary.
		resp.Diagnostics.AddError(handleLdapiErr(err).Error(), "")
		return
	}

	data.ID = types.StringValue(projectKey + "/" + key)
	data.Key = types.StringValue(pipeline.Key)
	data.Name = types.StringValue(pipeline.Name)
	if pipeline.Description != nil {
		data.Description = types.StringValue(*pipeline.Description)
	} else {
		data.Description = types.StringValue("")
	}
	data.CreatedAt = types.StringValue(fmt.Sprintf("%d", pipeline.CreatedAt.UnixMilli()))
	if pipeline.Version != nil {
		data.Version = types.Int64Value(int64(*pipeline.Version))
	} else {
		data.Version = types.Int64Value(0)
	}
	if pipeline.IsProjectDefault != nil {
		data.IsProjectDefault = types.BoolValue(*pipeline.IsProjectDefault)
	} else {
		data.IsProjectDefault = types.BoolValue(false)
	}

	tagsSet, diags := setFromStringSlice(ctx, pipeline.Tags)
	resp.Diagnostics.Append(diags...)
	data.Tags = tagsSet

	// Pass a null plan so the data source populates everything from the API.
	phasesList, diags := releasePipelinePhasesToList(ctx, pipeline.Phases, types.ListNull(types.ObjectType{AttrTypes: releasePipelinePhaseAttrTypes}))
	resp.Diagnostics.Append(diags...)
	data.Phases = phasesList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
