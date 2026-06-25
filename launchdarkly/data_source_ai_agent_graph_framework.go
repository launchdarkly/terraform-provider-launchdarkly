package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var _ datasource.DataSource = &AIAgentGraphDataSource{}

type AIAgentGraphDataSource struct {
	client *Client
}

type AIAgentGraphDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	ProjectKey        types.String `tfsdk:"project_key"`
	Key               types.String `tfsdk:"key"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	MaintainerID      types.String `tfsdk:"maintainer_id"`
	MaintainerTeamKey types.String `tfsdk:"maintainer_team_key"`
	RootConfigKey     types.String `tfsdk:"root_config_key"`
	Edges             types.List   `tfsdk:"edges"`
	CreationDate      types.Int64  `tfsdk:"creation_date"`
	LastModified      types.Int64  `tfsdk:"last_modified"`
}

func NewAIAgentGraphDataSource() datasource.DataSource {
	return &AIAgentGraphDataSource{}
}

func (d *AIAgentGraphDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_agent_graph"
}

func (d *AIAgentGraphDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly AI agent graph data source.\n\nThis data source allows you to retrieve information about an agent graph (a directed graph of AI Configs) in your LaunchDarkly project.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, Description: "The composite ID of the agent graph in the format `project_key/key`."},
			PROJECT_KEY:   schema.StringAttribute{Required: true, Description: "The project key."},
			KEY:           schema.StringAttribute{Required: true, Description: "The unique key of the agent graph."},
			NAME:          schema.StringAttribute{Computed: true, Description: "A human-readable name for the agent graph."},
			DESCRIPTION:   schema.StringAttribute{Computed: true, Description: "A description of the agent graph."},
			MAINTAINER_ID: schema.StringAttribute{Computed: true, Description: "The member ID of the maintainer for this agent graph."},
			MAINTAINER_TEAM_KEY: schema.StringAttribute{
				Computed:    true,
				Description: "The team key of the maintainer team for this agent graph.",
			},
			ROOT_CONFIG_KEY: schema.StringAttribute{Computed: true, Description: "The AI Config key of the root node of the graph."},
			EDGES: schema.ListNestedAttribute{
				Computed:    true,
				Description: "The edges in the graph. Each edge connects a source AI Config to a target AI Config.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						KEY:           schema.StringAttribute{Computed: true, Description: "A unique key for this edge within the graph."},
						SOURCE_CONFIG: schema.StringAttribute{Computed: true, Description: "The AI Config key that is the source of this edge."},
						TARGET_CONFIG: schema.StringAttribute{Computed: true, Description: "The AI Config key that is the target of this edge."},
						HANDOFF:       schema.StringAttribute{Computed: true, Description: "A JSON string representing the handoff options from the source AI Config to the target AI Config."},
					},
				},
			},
			CREATION_DATE: schema.Int64Attribute{Computed: true, Description: "The creation timestamp of the agent graph, in Unix epoch milliseconds."},
			LAST_MODIFIED: schema.Int64Attribute{Computed: true, Description: "The timestamp of the agent graph's last update, in Unix epoch milliseconds."},
		},
	}
}

func (d *AIAgentGraphDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *AIAgentGraphDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data AIAgentGraphDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	graphKey := data.Key.ValueString()

	var graph *ldapi.AgentGraph
	var err error
	err = d.client.withConcurrency(d.client.ctx, func() error {
		graph, _, err = d.client.ld.AIConfigsApi.GetAgentGraph(d.client.ctx, projectKey, graphKey).Execute()
		return err
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to get agent graph with key %q in project %q: %s", graphKey, projectKey, handleLdapiErr(err).Error()),
			"",
		)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", projectKey, graph.GetKey()))
	data.Key = types.StringValue(graph.GetKey())
	data.Name = types.StringValue(graph.GetName())
	data.Description = types.StringValue(graph.GetDescription())
	data.RootConfigKey = types.StringValue(graph.GetRootConfigKey())
	data.CreationDate = types.Int64Value(graph.GetCreatedAt())
	data.LastModified = types.Int64Value(graph.GetUpdatedAt())

	data.MaintainerID = types.StringValue("")
	data.MaintainerTeamKey = types.StringValue("")
	maintainer := graph.GetMaintainer()
	if maintainer.MaintainerMember != nil {
		data.MaintainerID = types.StringValue(maintainer.MaintainerMember.GetId())
	}
	if maintainer.AiConfigsMaintainerTeam != nil {
		data.MaintainerTeamKey = types.StringValue(maintainer.AiConfigsMaintainerTeam.GetKey())
	}

	edgeModels, err := agentGraphEdgeModelsFromAPI(graph.Edges)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read agent graph edges", err.Error())
		return
	}
	edgesList, diag := types.ListValueFrom(ctx, agentGraphEdgeObjectType(), edgeModels)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Edges = edgesList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
