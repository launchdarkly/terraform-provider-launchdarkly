package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v23"
)

var _ datasource.DataSource = &AIConfigDataSource{}

type AIConfigDataSource struct {
	client *Client
}

type AIConfigDataSourceModel struct {
	ID                  types.String `tfsdk:"id"`
	ProjectKey          types.String `tfsdk:"project_key"`
	Key                 types.String `tfsdk:"key"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	Mode                types.String `tfsdk:"mode"`
	Tags                types.Set    `tfsdk:"tags"`
	MaintainerID        types.String `tfsdk:"maintainer_id"`
	MaintainerTeamKey   types.String `tfsdk:"maintainer_team_key"`
	EvaluationMetricKey types.String `tfsdk:"evaluation_metric_key"`
	IsInverted          types.Bool   `tfsdk:"is_inverted"`
	Version             types.Int64  `tfsdk:"version"`
	CreationDate        types.Int64  `tfsdk:"creation_date"`
	Variations          types.List   `tfsdk:"variations"`
}

var aiConfigVariationSummaryAttrTypes = map[string]attr.Type{
	KEY:          types.StringType,
	NAME:         types.StringType,
	VARIATION_ID: types.StringType,
}

func NewAIConfigDataSource() datasource.DataSource {
	return &AIConfigDataSource{}
}

func (d *AIConfigDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_config"
}

func (d *AIConfigDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly AI Config data source.\n\nThis data source allows you to retrieve AI configuration information from your LaunchDarkly project.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, Description: "The ID in the format `project_key/key`."},
			PROJECT_KEY:   schema.StringAttribute{Required: true, Description: "The project key."},
			KEY:           schema.StringAttribute{Required: true, Description: "The AI Config's unique key."},
			NAME:          schema.StringAttribute{Computed: true, Description: "The AI Config's human-readable name."},
			DESCRIPTION:   schema.StringAttribute{Computed: true, Description: "The AI Config's description."},
			MODE:          schema.StringAttribute{Computed: true, Description: "The AI Config's mode. Must be `completion`, `agent`, or `judge`."},
			TAGS:          schema.SetAttribute{Computed: true, ElementType: types.StringType, Description: "Tags associated with your resource."},
			MAINTAINER_ID: schema.StringAttribute{Computed: true, Description: "The member ID of the maintainer for this AI Config. Conflicts with `maintainer_team_key`."},
			MAINTAINER_TEAM_KEY: schema.StringAttribute{
				Computed:    true,
				Description: "The team key of the maintainer team for this AI Config. Conflicts with `maintainer_id`.",
			},
			EVALUATION_METRIC_KEY: schema.StringAttribute{Computed: true, Description: "The key of the evaluation metric associated with this AI Config."},
			IS_INVERTED:           schema.BoolAttribute{Computed: true, Description: "Whether the evaluation metric is inverted."},
			VERSION:               schema.Int64Attribute{Computed: true, Description: "The version of the AI Config."},
			CREATION_DATE:         schema.Int64Attribute{Computed: true, Description: "A timestamp of when the AI Config was created."},
			VARIATIONS: schema.ListNestedAttribute{
				Computed:    true,
				Description: "A list of variation summaries for this AI Config.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						KEY:          schema.StringAttribute{Computed: true, Description: "The variation's key."},
						NAME:         schema.StringAttribute{Computed: true, Description: "The variation's name."},
						VARIATION_ID: schema.StringAttribute{Computed: true, Description: "The variation's ID."},
					},
				},
			},
		},
	}
}

func (d *AIConfigDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *AIConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data AIConfigDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	key := data.Key.ValueString()

	var aiConfig *ldapi.AIConfig
	var err error
	err = d.client.withConcurrency(d.client.ctx, func() error {
		aiConfig, _, err = d.client.ld.AgentControlApi.GetAIConfig(d.client.ctx, projectKey, key).Execute()
		return err
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to get AI config with key %q in project %q: %s", key, projectKey, handleLdapiErr(err).Error()),
			"",
		)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", projectKey, aiConfig.Key))
	data.Key = types.StringValue(aiConfig.Key)
	data.Name = types.StringValue(aiConfig.Name)
	data.Description = types.StringValue(aiConfig.Description)
	data.Version = types.Int64Value(int64(aiConfig.Version))
	data.CreationDate = types.Int64Value(aiConfig.CreatedAt)

	mode := "completion"
	if aiConfig.Mode != nil {
		mode = *aiConfig.Mode
	}
	data.Mode = types.StringValue(mode)

	if aiConfig.EvaluationMetricKey != nil {
		data.EvaluationMetricKey = types.StringValue(*aiConfig.EvaluationMetricKey)
	} else {
		data.EvaluationMetricKey = types.StringValue("")
	}
	if aiConfig.IsInverted != nil {
		data.IsInverted = types.BoolValue(*aiConfig.IsInverted)
	} else {
		data.IsInverted = types.BoolValue(false)
	}

	data.MaintainerID = types.StringValue("")
	data.MaintainerTeamKey = types.StringValue("")
	maintainer := aiConfig.GetMaintainer()
	if maintainer.MaintainerMember != nil {
		data.MaintainerID = types.StringValue(maintainer.MaintainerMember.GetId())
	}
	if maintainer.AiConfigsMaintainerTeam != nil {
		data.MaintainerTeamKey = types.StringValue(maintainer.AiConfigsMaintainerTeam.GetKey())
	}

	tagsSet, diags := setFromStringSlice(ctx, aiConfig.Tags)
	resp.Diagnostics.Append(diags...)
	data.Tags = tagsSet

	objectType := types.ObjectType{AttrTypes: aiConfigVariationSummaryAttrTypes}
	elements := make([]attr.Value, 0, len(aiConfig.Variations))
	for _, v := range aiConfig.Variations {
		obj, d := types.ObjectValue(aiConfigVariationSummaryAttrTypes, map[string]attr.Value{
			KEY:          types.StringValue(v.Key),
			NAME:         types.StringValue(v.Name),
			VARIATION_ID: types.StringValue(v.Id),
		})
		resp.Diagnostics.Append(d...)
		elements = append(elements, obj)
	}
	list, diags := types.ListValue(objectType, elements)
	resp.Diagnostics.Append(diags...)
	data.Variations = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
