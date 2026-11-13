package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &EnvironmentDefaultReleasePlanResource{}
	_ resource.ResourceWithImportState = &EnvironmentDefaultReleasePlanResource{}
)

type EnvironmentDefaultReleasePlanResource struct {
	client *Client
}

type EnvironmentDefaultReleasePlanResourceModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectKey     types.String `tfsdk:"project_key"`
	EnvironmentKey types.String `tfsdk:"environment_key"`
	ReleasePlanKey types.String `tfsdk:"release_plan_key"`
}

func NewEnvironmentDefaultReleasePlanResource() resource.Resource {
	return &EnvironmentDefaultReleasePlanResource{}
}

func (r *EnvironmentDefaultReleasePlanResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_default_release_plan"
}

func (r *EnvironmentDefaultReleasePlanResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Provides a LaunchDarkly environment default release plan resource.

~> **Beta:** This resource wraps a beta LaunchDarkly API (the ` + "`default-release-plan`" + ` endpoint, accessed with the ` + "`LD-API-Version: beta`" + ` header). Beta resources may change or be removed in future versions.

This resource sets which [release plan](https://launchdarkly.com/docs/home/releases) is the default for a LaunchDarkly environment. An environment has at most one default release plan, so declare at most one of these resources per environment. Setting a new default replaces the previous one. The plan must be an active release plan in the environment and must have a randomization unit.

-> **Note:** Destroying this resource clears the environment's default release plan. The release plan itself is not deleted.`,
		Attributes: map[string]schema.Attribute{
			ID: schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of this resource in the format `project_key/environment_key`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:      true,
				Description:   addForceNewDescription("The project key of the environment.", true),
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			ENVIRONMENT_KEY: schema.StringAttribute{
				Required:      true,
				Description:   addForceNewDescription("The key of the environment whose default release plan this resource manages.", true),
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			RELEASE_PLAN_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The key of the release plan to use as the environment's default. Reference a `launchdarkly_environment_release_plan` resource's `key` so Terraform creates the plan first.",
				Validators:  []validator.String{keyValidator()},
			},
		},
	}
}

func (r *EnvironmentDefaultReleasePlanResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *EnvironmentDefaultReleasePlanResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EnvironmentDefaultReleasePlanResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.setDefault(ctx, &plan, "creating", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EnvironmentDefaultReleasePlanResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data EnvironmentDefaultReleasePlanResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	envKey := data.EnvironmentKey.ValueString()

	plan, res, err := getEnvironmentDefaultReleasePlan(ctx, r.client, projectKey, envKey)
	if err != nil {
		if isStatusNotFound(res) {
			resp.State.RemoveResource(ctx)
			return
		}
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Failed to get default release plan for environment %q of project %q", envKey, projectKey), err)
		return
	}
	// A 204 means the environment has no default any more.
	if plan == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	environmentDefaultReleasePlanToModel(projectKey, envKey, plan, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EnvironmentDefaultReleasePlanResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EnvironmentDefaultReleasePlanResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.setDefault(ctx, &plan, "updating", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EnvironmentDefaultReleasePlanResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data EnvironmentDefaultReleasePlanResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	envKey := data.EnvironmentKey.ValueString()
	res, err := clearEnvironmentDefaultReleasePlan(ctx, r.client, projectKey, envKey)
	if err != nil {
		if isStatusNotFound(res) {
			return
		}
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error clearing default release plan for environment %q of project %q", envKey, projectKey), err)
	}
}

func (r *EnvironmentDefaultReleasePlanResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectKey, envKey, err := environmentDefaultReleasePlanIdToKeys(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), projectKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(ENVIRONMENT_KEY), envKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(ID), req.ID)...)
}

// setDefault PUTs the planned key and reads the default back so state reflects
// what the API actually stored. The PUT response body is not relied on.
func (r *EnvironmentDefaultReleasePlanResource) setDefault(ctx context.Context, plan *EnvironmentDefaultReleasePlanResourceModel, verb string, diags diagnosticsSink) {
	projectKey := plan.ProjectKey.ValueString()
	envKey := plan.EnvironmentKey.ValueString()
	planKey := plan.ReleasePlanKey.ValueString()

	if err := setEnvironmentDefaultReleasePlan(ctx, r.client, projectKey, envKey, planKey); err != nil {
		addLdapiError(diags, fmt.Sprintf("Error %s default release plan %q for environment %q of project %q", verb, planKey, envKey, projectKey), err)
		return
	}

	current, _, err := getEnvironmentDefaultReleasePlan(ctx, r.client, projectKey, envKey)
	if err != nil {
		addLdapiError(diags, fmt.Sprintf("Failed to read back default release plan for environment %q of project %q", envKey, projectKey), err)
		return
	}
	if current == nil {
		diags.AddError(
			fmt.Sprintf("Error %s default release plan for environment %q of project %q", verb, envKey, projectKey),
			fmt.Sprintf("The API accepted release plan %q as the default but reports no default is set.", planKey),
		)
		return
	}
	environmentDefaultReleasePlanToModel(projectKey, envKey, current, plan)
}

func environmentDefaultReleasePlanToModel(projectKey, envKey string, rep *environmentReleasePlanRep, m *EnvironmentDefaultReleasePlanResourceModel) {
	m.ID = types.StringValue(environmentDefaultReleasePlanID(projectKey, envKey))
	m.ProjectKey = types.StringValue(projectKey)
	m.EnvironmentKey = types.StringValue(envKey)
	m.ReleasePlanKey = types.StringValue(rep.Key)
}
