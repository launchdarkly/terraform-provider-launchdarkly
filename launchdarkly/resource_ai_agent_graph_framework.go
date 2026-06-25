package launchdarkly

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var (
	_ resource.Resource                     = &AIAgentGraphResource{}
	_ resource.ResourceWithImportState      = &AIAgentGraphResource{}
	_ resource.ResourceWithConfigValidators = &AIAgentGraphResource{}
)

type AIAgentGraphResource struct {
	client *Client
}

type AIAgentGraphResourceModel struct {
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

func NewAIAgentGraphResource() resource.Resource {
	return &AIAgentGraphResource{}
}

func (r *AIAgentGraphResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_agent_graph"
}

func (r *AIAgentGraphResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly AI agent graph resource.\n\nAn agent graph represents a directed graph of AI Configs, connecting them with edges that describe handoffs from one AI Config to another. This resource allows you to create and manage agent graphs within a LaunchDarkly project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The composite ID of the agent graph in the format `project_key/key`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The project key. A change in this field will force the destruction of the existing resource and the creation of a new one.",
				Validators:  []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			KEY: schema.StringAttribute{
				Required:    true,
				Description: "The unique key of the agent graph. A change in this field will force the destruction of the existing resource and the creation of a new one.",
				Validators:  []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			NAME: schema.StringAttribute{
				Required:    true,
				Description: "A human-readable name for the agent graph.",
			},
			DESCRIPTION: schema.StringAttribute{
				Optional:    true,
				Description: "A description of the agent graph.",
			},
			MAINTAINER_ID: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The member ID of the maintainer for this agent graph. Defaults to the member associated with the access token. Conflicts with `maintainer_team_key`.",
				Validators:  []validator.String{idValidator()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			MAINTAINER_TEAM_KEY: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The team key of the maintainer team for this agent graph. Conflicts with `maintainer_id`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			ROOT_CONFIG_KEY: schema.StringAttribute{
				Optional:    true,
				Description: "The AI Config key of the root node of the graph. If `root_config_key` or `edges` is set, both must be set. A graph with neither defined is a metadata-only graph.",
			},
			EDGES: schema.ListNestedAttribute{
				Optional:    true,
				Description: "The edges in the graph. Each edge connects a source AI Config to a target AI Config. If `edges` or `root_config_key` is set, both must be set.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						KEY: schema.StringAttribute{
							Required:    true,
							Description: "A unique key for this edge within the graph.",
							Validators:  []validator.String{keyValidator()},
						},
						SOURCE_CONFIG: schema.StringAttribute{
							Required:    true,
							Description: "The AI Config key that is the source of this edge.",
							Validators:  []validator.String{keyValidator()},
						},
						TARGET_CONFIG: schema.StringAttribute{
							Required:    true,
							Description: "The AI Config key that is the target of this edge.",
							Validators:  []validator.String{keyValidator()},
						},
						HANDOFF: schema.StringAttribute{
							Optional:    true,
							Description: "A JSON string representing the handoff options from the source AI Config to the target AI Config.",
							Validators:  []validator.String{jsonStringValidator{}},
							PlanModifiers: []planmodifier.String{
								jsonNormalizePlanModifier{},
							},
						},
					},
				},
			},
			CREATION_DATE: schema.Int64Attribute{
				Computed:    true,
				Description: "The creation timestamp of the agent graph, in Unix epoch milliseconds.",
			},
			LAST_MODIFIED: schema.Int64Attribute{
				Computed:    true,
				Description: "The timestamp of the agent graph's last update, in Unix epoch milliseconds.",
			},
		},
	}
}

func (r *AIAgentGraphResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		agentGraphMaintainerValidator{},
		agentGraphRootEdgesValidator{},
	}
}

// agentGraphMaintainerValidator enforces that maintainer_id and
// maintainer_team_key are not set together.
type agentGraphMaintainerValidator struct{}

func (agentGraphMaintainerValidator) Description(context.Context) string {
	return "maintainer_id and maintainer_team_key are mutually exclusive"
}

func (agentGraphMaintainerValidator) MarkdownDescription(context.Context) string { return "" }

func (agentGraphMaintainerValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data AIAgentGraphResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	idSet := !data.MaintainerID.IsNull() && !data.MaintainerID.IsUnknown() && data.MaintainerID.ValueString() != ""
	teamSet := !data.MaintainerTeamKey.IsNull() && !data.MaintainerTeamKey.IsUnknown() && data.MaintainerTeamKey.ValueString() != ""
	if idSet && teamSet {
		resp.Diagnostics.AddAttributeError(
			path.Root(MAINTAINER_TEAM_KEY),
			"Conflicting maintainer fields",
			"maintainer_id and maintainer_team_key are mutually exclusive; set only one.",
		)
	}
}

// agentGraphRootEdgesValidator enforces the API constraint that root_config_key
// and edges must both be set or both be unset.
type agentGraphRootEdgesValidator struct{}

func (agentGraphRootEdgesValidator) Description(context.Context) string {
	return "root_config_key and edges must both be set or both be unset"
}

func (agentGraphRootEdgesValidator) MarkdownDescription(context.Context) string { return "" }

func (agentGraphRootEdgesValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data AIAgentGraphResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rootSet := !data.RootConfigKey.IsNull() && !data.RootConfigKey.IsUnknown() && data.RootConfigKey.ValueString() != ""
	edgesSet := !data.Edges.IsNull() && !data.Edges.IsUnknown() && len(data.Edges.Elements()) > 0
	if rootSet != edgesSet {
		resp.Diagnostics.AddError(
			"Incomplete agent graph definition",
			"`root_config_key` and `edges` must both be set or both be unset. Set both to define a graph with nodes, or neither for a metadata-only graph.",
		)
	}
}

func (r *AIAgentGraphResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *AIAgentGraphResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AIAgentGraphResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	graphKey := plan.Key.ValueString()

	post := ldapi.NewAgentGraphPost(graphKey, plan.Name.ValueString())
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		post.Description = ldapi.PtrString(plan.Description.ValueString())
	}
	if !plan.MaintainerID.IsNull() && !plan.MaintainerID.IsUnknown() && plan.MaintainerID.ValueString() != "" {
		post.MaintainerId = ldapi.PtrString(plan.MaintainerID.ValueString())
	}
	if !plan.MaintainerTeamKey.IsNull() && !plan.MaintainerTeamKey.IsUnknown() && plan.MaintainerTeamKey.ValueString() != "" {
		post.MaintainerTeamKey = ldapi.PtrString(plan.MaintainerTeamKey.ValueString())
	}
	if !plan.RootConfigKey.IsNull() && !plan.RootConfigKey.IsUnknown() && plan.RootConfigKey.ValueString() != "" {
		post.RootConfigKey = ldapi.PtrString(plan.RootConfigKey.ValueString())
	}
	if !plan.Edges.IsNull() && !plan.Edges.IsUnknown() {
		var edgeModels []agentGraphEdgeModel
		resp.Diagnostics.Append(plan.Edges.ElementsAs(ctx, &edgeModels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		edges, err := agentGraphEdgePostsFromModel(edgeModels)
		if err != nil {
			resp.Diagnostics.AddError("Invalid edges", err.Error())
			return
		}
		post.Edges = edges
	}

	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, _, err := r.client.ld.AIConfigsApi.PostAgentGraph(r.client.ctx, projectKey).AgentGraphPost(*post).Execute()
		return err
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Failed to create agent graph %q in project %q", graphKey, projectKey), err)
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/%s", projectKey, graphKey))
	r.readIntoModel(ctx, projectKey, graphKey, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AIAgentGraphResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AIAgentGraphResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	graphKey := data.Key.ValueString()
	r.readIntoModel(ctx, projectKey, graphKey, &data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AIAgentGraphResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AIAgentGraphResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state AIAgentGraphResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	graphKey := plan.Key.ValueString()

	// The PATCH endpoint accepts a JSON merge patch (not a JSON patch array):
	// each field present in the body is replaced server-side.
	patch := ldapi.NewAgentGraphPatch()
	if !plan.Name.Equal(state.Name) {
		patch.Name = ldapi.PtrString(plan.Name.ValueString())
	}
	if !plan.Description.Equal(state.Description) {
		patch.Description = ldapi.PtrString(plan.Description.ValueString())
	}
	// Use ValueString() so removing a maintainer from config sends an empty
	// string, which the API treats as "remove maintainer".
	if !plan.MaintainerID.Equal(state.MaintainerID) {
		patch.MaintainerId = ldapi.PtrString(plan.MaintainerID.ValueString())
	}
	if !plan.MaintainerTeamKey.Equal(state.MaintainerTeamKey) {
		patch.MaintainerTeamKey = ldapi.PtrString(plan.MaintainerTeamKey.ValueString())
	}
	// root_config_key and edges must travel together; if either changed, send
	// both at their planned values (the config validator guarantees they are
	// both set or both unset).
	if !plan.RootConfigKey.Equal(state.RootConfigKey) || !plan.Edges.Equal(state.Edges) {
		patch.RootConfigKey = ldapi.PtrString(plan.RootConfigKey.ValueString())
		if !plan.Edges.IsNull() && !plan.Edges.IsUnknown() {
			var edgeModels []agentGraphEdgeModel
			resp.Diagnostics.Append(plan.Edges.ElementsAs(ctx, &edgeModels, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
			edges, err := agentGraphEdgesFromModel(edgeModels)
			if err != nil {
				resp.Diagnostics.AddError("Invalid edges", err.Error())
				return
			}
			patch.Edges = edges
		}
	}

	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, _, err := r.client.ld.AIConfigsApi.PatchAgentGraph(r.client.ctx, projectKey, graphKey).AgentGraphPatch(*patch).Execute()
		return err
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Failed to update agent graph %q in project %q", graphKey, projectKey), err)
		return
	}

	r.readIntoModel(ctx, projectKey, graphKey, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AIAgentGraphResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AIAgentGraphResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectKey := data.ProjectKey.ValueString()
	graphKey := data.Key.ValueString()
	var res *http.Response
	err := r.client.withConcurrency(r.client.ctx, func() error {
		var e error
		res, e = r.client.ld.AIConfigsApi.DeleteAgentGraph(r.client.ctx, projectKey, graphKey).Execute()
		return e
	})
	if err != nil {
		if isStatusNotFound(res) {
			return
		}
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Failed to delete agent graph %q in project %q", graphKey, projectKey), err)
	}
}

func (r *AIAgentGraphResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Invalid import ID", "import ID cannot be empty")
		return
	}
	projectKey, graphKey, err := aiAgentGraphIDToKeys(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), projectKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(KEY), graphKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *AIAgentGraphResource) readIntoModel(
	ctx context.Context,
	projectKey, graphKey string,
	data *AIAgentGraphResourceModel,
	diags *diag.Diagnostics,
) {
	var graph *ldapi.AgentGraph
	var res *http.Response
	var err error
	err = r.client.withConcurrency(r.client.ctx, func() error {
		graph, res, err = r.client.ld.AIConfigsApi.GetAgentGraph(r.client.ctx, projectKey, graphKey).Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError("Failed to get agent graph", handleLdapiErr(err).Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", projectKey, graph.GetKey()))
	data.ProjectKey = types.StringValue(projectKey)
	data.Key = types.StringValue(graph.GetKey())
	data.Name = types.StringValue(graph.GetName())
	// Optional-only attr: null-when-empty for plan-apply consistency.
	data.Description = stringValueOrNullFromPointer(graph.Description)
	data.RootConfigKey = stringValueOrNullFromPointer(graph.RootConfigKey)
	data.CreationDate = types.Int64Value(graph.GetCreatedAt())
	data.LastModified = types.Int64Value(graph.GetUpdatedAt())

	// maintainer_id and maintainer_team_key are Computed; the API fills one of
	// them in based on the resolved maintainer.
	data.MaintainerID = types.StringValue("")
	data.MaintainerTeamKey = types.StringValue("")
	maintainer := graph.GetMaintainer()
	if maintainer.MaintainerMember != nil {
		data.MaintainerID = types.StringValue(maintainer.MaintainerMember.GetId())
	} else if maintainer.AiConfigsMaintainerTeam != nil {
		data.MaintainerTeamKey = types.StringValue(maintainer.AiConfigsMaintainerTeam.GetKey())
	}

	edgeModels, err := agentGraphEdgeModelsFromAPI(graph.Edges)
	if err != nil {
		diags.AddError("Failed to read agent graph edges", err.Error())
		return
	}
	if len(edgeModels) == 0 {
		// Optional-only attr: null-when-empty so a metadata-only graph stays null.
		data.Edges = types.ListNull(agentGraphEdgeObjectType())
		return
	}
	edgesList, d := types.ListValueFrom(ctx, agentGraphEdgeObjectType(), edgeModels)
	diags.Append(d...)
	data.Edges = edgesList
}
