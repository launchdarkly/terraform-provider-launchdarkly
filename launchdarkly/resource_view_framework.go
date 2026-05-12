package launchdarkly

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &ViewResource{}
	_ resource.ResourceWithImportState = &ViewResource{}
)

type ViewResource struct {
	client *Client
}

type ViewResourceModel struct {
	ID                types.String `tfsdk:"id"`
	ProjectKey        types.String `tfsdk:"project_key"`
	Key               types.String `tfsdk:"key"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	MaintainerID      types.String `tfsdk:"maintainer_id"`
	MaintainerTeamKey types.String `tfsdk:"maintainer_team_key"`
	Tags              types.Set    `tfsdk:"tags"`
	Archived          types.Bool   `tfsdk:"archived"`
}

func NewViewResource() resource.Resource {
	return &ViewResource{}
}

func (r *ViewResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_view"
}

func (r *ViewResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly view resource (Enterprise, beta API).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:      true,
				Description:   "The project key.",
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			KEY: schema.StringAttribute{
				Required:      true,
				Description:   "The view's unique key.",
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			NAME: schema.StringAttribute{
				Required:    true,
				Description: "View's name.",
			},
			DESCRIPTION: schema.StringAttribute{
				Optional:    true,
				Description: "View's description.",
			},
			MAINTAINER_ID: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Member ID of the maintainer. Exactly one of maintainer_id / maintainer_team_key must be set.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			MAINTAINER_TEAM_KEY: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Team key of the maintainer team.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			TAGS: schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Tags.",
			},
			ARCHIVED: schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the view is archived.",
			},
		},
	}
}

func (r *ViewResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{viewMaintainerExactlyOneValidator{}}
}

type viewMaintainerExactlyOneValidator struct{}

func (viewMaintainerExactlyOneValidator) Description(context.Context) string {
	return "exactly one of maintainer_id or maintainer_team_key must be set"
}
func (viewMaintainerExactlyOneValidator) MarkdownDescription(ctx context.Context) string {
	return ""
}
func (viewMaintainerExactlyOneValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ViewResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	idSet := !data.MaintainerID.IsNull() && !data.MaintainerID.IsUnknown() && data.MaintainerID.ValueString() != ""
	teamSet := !data.MaintainerTeamKey.IsNull() && !data.MaintainerTeamKey.IsUnknown() && data.MaintainerTeamKey.ValueString() != ""
	if idSet == teamSet {
		resp.Diagnostics.AddError(
			"Maintainer configuration",
			"Exactly one of maintainer_id or maintainer_team_key must be set.",
		)
	}
}

// viewExists is retained as a package-level helper because the
// view_links / view_filter_links SDKv2 resources still reference it.
// Moves with those resources in Phase 3.8.
func viewExists(projectKey, viewKey string, client *Client) (bool, error) {
	_, res, err := getViewRaw(client, projectKey, viewKey)
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get view %q in project %q: %s", viewKey, projectKey, handleLdapiErr(err))
	}
	return true, nil
}

func (r *ViewResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *ViewResource) betaClient() (*Client, error) {
	return newBetaClient(r.client.apiKey, r.client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
}

func (r *ViewResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ViewResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	beta, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	viewKey := plan.Key.ValueString()
	viewPost := map[string]interface{}{
		"key":  viewKey,
		"name": plan.Name.ValueString(),
	}
	if plan.Description.ValueString() != "" {
		viewPost["description"] = plan.Description.ValueString()
	}
	if plan.MaintainerID.ValueString() != "" {
		viewPost["maintainerId"] = plan.MaintainerID.ValueString()
	}
	if plan.MaintainerTeamKey.ValueString() != "" {
		viewPost["maintainerTeamKey"] = plan.MaintainerTeamKey.ValueString()
	}
	tags, diags := stringSliceFromSet(ctx, plan.Tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(tags) > 0 {
		viewPost["tags"] = tags
	}

	if _, err := createView(beta, projectKey, viewPost); err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to create view", err)
		return
	}
	plan.ID = types.StringValue(fmt.Sprintf("%s/%s", projectKey, viewKey))

	r.readIntoModel(ctx, projectKey, viewKey, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ViewResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ViewResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readIntoModel(ctx, data.ProjectKey.ValueString(), data.Key.ValueString(), &data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ViewResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ViewResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	beta, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}

	patch := map[string]interface{}{}
	if !plan.Name.Equal(state.Name) {
		patch["name"] = plan.Name.ValueString()
	}
	if !plan.Description.Equal(state.Description) {
		patch["description"] = plan.Description.ValueString()
	}
	if !plan.MaintainerID.Equal(state.MaintainerID) && plan.MaintainerID.ValueString() != "" {
		patch["maintainerId"] = plan.MaintainerID.ValueString()
	}
	if !plan.MaintainerTeamKey.Equal(state.MaintainerTeamKey) && plan.MaintainerTeamKey.ValueString() != "" {
		patch["maintainerTeamKey"] = plan.MaintainerTeamKey.ValueString()
	}
	if !plan.Tags.Equal(state.Tags) {
		tags, diags := stringSliceFromSet(ctx, plan.Tags)
		resp.Diagnostics.Append(diags...)
		patch["tags"] = tags
	}
	if !plan.Archived.Equal(state.Archived) {
		patch["archived"] = plan.Archived.ValueBool()
	}

	if len(patch) > 0 {
		if err := patchView(beta, plan.ProjectKey.ValueString(), plan.Key.ValueString(), patch); err != nil {
			addLdapiError(&resp.Diagnostics, "Failed to update view", err)
			return
		}
	}

	r.readIntoModel(ctx, plan.ProjectKey.ValueString(), plan.Key.ValueString(), &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ViewResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ViewResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	beta, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}
	if err := deleteView(beta, data.ProjectKey.ValueString(), data.Key.ValueString()); err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to delete view", err)
	}
}

func (r *ViewResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "expected project_key/view_key")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(KEY), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *ViewResource) readIntoModel(
	ctx context.Context,
	projectKey, viewKey string,
	data *ViewResourceModel,
	diags interface{ AddError(string, string) },
) {
	beta, err := r.betaClient()
	if err != nil {
		diags.AddError("Failed to build beta client", err.Error())
		return
	}
	view, res, err := getView(beta, projectKey, viewKey)
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError("Failed to get view", handleLdapiErr(err).Error())
		return
	}
	data.ID = types.StringValue(fmt.Sprintf("%s/%s", projectKey, viewKey))
	data.ProjectKey = types.StringValue(view.ProjectKey)
	data.Key = types.StringValue(view.Key)
	data.Name = types.StringValue(view.Name)
	if view.Description != nil {
		data.Description = types.StringValue(*view.Description)
	} else {
		data.Description = types.StringValue("")
	}
	if view.Archived != nil {
		data.Archived = types.BoolValue(*view.Archived)
	} else {
		data.Archived = types.BoolValue(false)
	}

	data.MaintainerID = types.StringValue("")
	data.MaintainerTeamKey = types.StringValue("")
	if view.Maintainer != nil {
		switch view.Maintainer.Kind {
		case "member":
			if view.Maintainer.MaintainerMember != nil {
				data.MaintainerID = types.StringValue(view.Maintainer.MaintainerMember.Id)
			}
		case "team":
			if view.Maintainer.MaintainerTeam != nil {
				data.MaintainerTeamKey = types.StringValue(view.Maintainer.MaintainerTeam.Key)
			}
		}
	}

	tagsSet, d := setFromStringSlice(ctx, view.Tags)
	if d.HasError() {
		for _, e := range d.Errors() {
			diags.AddError(e.Summary(), e.Detail())
		}
	}
	data.Tags = tagsSet
}
