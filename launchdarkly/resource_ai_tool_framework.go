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
	_ resource.Resource                = &AIToolResource{}
	_ resource.ResourceWithImportState = &AIToolResource{}
)

type AIToolResource struct {
	client *Client
}

type AIToolResourceModel struct {
	ID                types.String `tfsdk:"id"`
	ProjectKey        types.String `tfsdk:"project_key"`
	Key               types.String `tfsdk:"key"`
	Description       types.String `tfsdk:"description"`
	SchemaJSON        types.String `tfsdk:"schema_json"`
	CustomParameters  types.String `tfsdk:"custom_parameters"`
	MaintainerID      types.String `tfsdk:"maintainer_id"`
	MaintainerTeamKey types.String `tfsdk:"maintainer_team_key"`
	Version           types.Int64  `tfsdk:"version"`
	CreationDate      types.Int64  `tfsdk:"creation_date"`
}

func NewAIToolResource() resource.Resource {
	return &AIToolResource{}
}

func (r *AIToolResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_tool"
}

func (r *AIToolResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly AI tool resource.\n\nThis resource allows you to create and manage AI tools within your LaunchDarkly project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Composite ID `project_key/key`.",
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
				Description: "The AI tool's unique key. A change in this field will force the destruction of the existing resource and the creation of a new one.",
				Validators:  []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			DESCRIPTION: schema.StringAttribute{
				Optional:    true,
				Description: "The AI tool's description.",
			},
			SCHEMA_JSON: schema.StringAttribute{
				Required:    true,
				Description: "A JSON string representing the JSON Schema for the tool's parameters.",
				Validators:  []validator.String{jsonSchemaStringValidator{}},
				PlanModifiers: []planmodifier.String{
					jsonNormalizePlanModifier{},
				},
			},
			CUSTOM_PARAMETERS: schema.StringAttribute{
				Optional:    true,
				Description: "A JSON string representing custom application-level metadata for the AI tool.",
				Validators:  []validator.String{jsonStringValidator{}},
				PlanModifiers: []planmodifier.String{
					jsonNormalizePlanModifier{},
				},
			},
			MAINTAINER_ID: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The member ID of the maintainer for this AI tool. Conflicts with `maintainer_team_key`.",
				Validators:  []validator.String{idValidator()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			MAINTAINER_TEAM_KEY: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The team key of the maintainer team for this AI tool. Conflicts with `maintainer_id`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			VERSION:       schema.Int64Attribute{Computed: true, Description: "The version of the AI tool."},
			CREATION_DATE: schema.Int64Attribute{Computed: true, Description: "The creation timestamp of the AI tool."},
		},
	}
}

func (r *AIToolResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		// maintainer_id and maintainer_team_key are mutually exclusive.
		conflictingMaintainerValidator{},
	}
}

// conflictingMaintainerValidator enforces that maintainer_id and
// maintainer_team_key are not set together.
type conflictingMaintainerValidator struct{}

func (conflictingMaintainerValidator) Description(context.Context) string {
	return "maintainer_id and maintainer_team_key are mutually exclusive"
}

func (conflictingMaintainerValidator) MarkdownDescription(ctx context.Context) string {
	return ""
}

func (conflictingMaintainerValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data AIToolResourceModel
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

func (r *AIToolResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *AIToolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AIToolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	toolKey := plan.Key.ValueString()

	schemaMap, err := jsonStringToMap(plan.SchemaJSON.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid schema_json", err.Error())
		return
	}

	post := ldapi.NewAIToolPost(toolKey, schemaMap)
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		post.Description = ldapi.PtrString(plan.Description.ValueString())
	}
	if !plan.CustomParameters.IsNull() && !plan.CustomParameters.IsUnknown() {
		customParamsMap, err := jsonStringToMap(plan.CustomParameters.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid custom_parameters", err.Error())
			return
		}
		post.CustomParameters = customParamsMap
	}
	if !plan.MaintainerID.IsNull() && !plan.MaintainerID.IsUnknown() && plan.MaintainerID.ValueString() != "" {
		post.MaintainerId = ldapi.PtrString(plan.MaintainerID.ValueString())
	}
	if !plan.MaintainerTeamKey.IsNull() && !plan.MaintainerTeamKey.IsUnknown() && plan.MaintainerTeamKey.ValueString() != "" {
		post.MaintainerTeamKey = ldapi.PtrString(plan.MaintainerTeamKey.ValueString())
	}

	err = r.client.withConcurrency(r.client.ctx, func() error {
		_, _, err := r.client.ld.AIConfigsApi.PostAITool(r.client.ctx, projectKey).AIToolPost(*post).Execute()
		return err
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to create AI tool", err)
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/%s", projectKey, toolKey))
	r.readIntoModel(ctx, projectKey, toolKey, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AIToolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AIToolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	key := data.Key.ValueString()
	r.readIntoModel(ctx, projectKey, key, &data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AIToolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AIToolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state AIToolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	toolKey := plan.Key.ValueString()

	patch := ldapi.NewAIToolPatch()
	if !plan.Description.Equal(state.Description) {
		v := plan.Description.ValueString()
		patch.Description = &v
	}
	if !plan.SchemaJSON.Equal(state.SchemaJSON) {
		m, err := jsonStringToMap(plan.SchemaJSON.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid schema_json", err.Error())
			return
		}
		patch.Schema = m
	}
	if !plan.CustomParameters.Equal(state.CustomParameters) {
		m, err := jsonStringToMap(plan.CustomParameters.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid custom_parameters", err.Error())
			return
		}
		patch.CustomParameters = m
	}
	// Use ValueString() so that removing a maintainer from config sends an
	// empty string to the API, clearing it server-side — matching SDKv2 behaviour
	// which uses d.Get() instead of d.GetOk().
	if !plan.MaintainerID.Equal(state.MaintainerID) {
		v := plan.MaintainerID.ValueString()
		patch.MaintainerId = &v
	}
	if !plan.MaintainerTeamKey.Equal(state.MaintainerTeamKey) {
		v := plan.MaintainerTeamKey.ValueString()
		patch.MaintainerTeamKey = &v
	}

	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, _, err := r.client.ld.AIConfigsApi.PatchAITool(r.client.ctx, projectKey, toolKey).AIToolPatch(*patch).Execute()
		return err
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to update AI tool", err)
		return
	}

	r.readIntoModel(ctx, projectKey, toolKey, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AIToolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AIToolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectKey := data.ProjectKey.ValueString()
	toolKey := data.Key.ValueString()
	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, err := r.client.ld.AIConfigsApi.DeleteAITool(r.client.ctx, projectKey, toolKey).Execute()
		return err
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to delete AI tool", err)
	}
}

func (r *AIToolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Invalid import ID", "import ID cannot be empty")
		return
	}
	projectKey, toolKey, err := aiToolIdToKeys(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), projectKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(KEY), toolKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *AIToolResource) readIntoModel(
	ctx context.Context,
	projectKey, toolKey string,
	data *AIToolResourceModel,
	diags *diag.Diagnostics,
) {
	var tool *ldapi.AITool
	var res *http.Response
	var err error
	err = r.client.withConcurrency(r.client.ctx, func() error {
		tool, res, err = r.client.ld.AIConfigsApi.GetAITool(r.client.ctx, projectKey, toolKey).Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError("Failed to get AI tool", handleLdapiErr(err).Error())
		return
	}
	data.ID = types.StringValue(fmt.Sprintf("%s/%s", projectKey, tool.GetKey()))
	data.ProjectKey = types.StringValue(projectKey)
	data.Key = types.StringValue(tool.GetKey())
	// Optional-only attr: null-when-empty for plan-apply consistency.
	data.Description = stringValueOrNullFromPointer(tool.Description)
	schemaJSON, err := mapToJsonString(tool.GetSchema())
	if err != nil {
		diags.AddError("Failed to serialise schema_json", err.Error())
		return
	}
	data.SchemaJSON = types.StringValue(schemaJSON)
	customParamsJSON, err := mapToJsonString(tool.GetCustomParameters())
	if err != nil {
		diags.AddError("Failed to serialise custom_parameters", err.Error())
		return
	}
	// Optional-only attr: null-when-empty so unset custom_parameters
	// doesn't trip terraform-core consistency check.
	data.CustomParameters = stringValueOrNull(customParamsJSON)
	data.Version = types.Int64Value(int64(tool.GetVersion()))
	data.CreationDate = types.Int64Value(tool.GetCreatedAt())

	data.MaintainerID = types.StringValue("")
	data.MaintainerTeamKey = types.StringValue("")
	if maintainer := tool.GetMaintainer(); maintainer.MaintainerMember != nil {
		data.MaintainerID = types.StringValue(maintainer.MaintainerMember.GetId())
	} else if tool.GetMaintainer().AiConfigsMaintainerTeam != nil {
		data.MaintainerTeamKey = types.StringValue(tool.GetMaintainer().AiConfigsMaintainerTeam.GetKey())
	}
}
