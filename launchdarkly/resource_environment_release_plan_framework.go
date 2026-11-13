package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &EnvironmentReleasePlanResource{}
	_ resource.ResourceWithImportState    = &EnvironmentReleasePlanResource{}
	_ resource.ResourceWithValidateConfig = &EnvironmentReleasePlanResource{}
)

type EnvironmentReleasePlanResource struct {
	client *Client
}

type EnvironmentReleasePlanResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ProjectKey         types.String `tfsdk:"project_key"`
	EnvironmentKey     types.String `tfsdk:"environment_key"`
	Key                types.String `tfsdk:"key"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	ReleaseMethod      types.String `tfsdk:"release_method"`
	Version            types.Int64  `tfsdk:"version"`
	IsDefault          types.Bool   `tfsdk:"is_default"`
	ProgressiveRelease types.Object `tfsdk:"progressive_release"`
}

func NewEnvironmentReleasePlanResource() resource.Resource { return &EnvironmentReleasePlanResource{} }

func (r *EnvironmentReleasePlanResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_release_plan"
}

func (r *EnvironmentReleasePlanResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Provides a LaunchDarkly environment release plan resource.

~> **Beta:** This resource wraps a beta LaunchDarkly API (the ` + "`release-plans`" + ` endpoints, accessed with the ` + "`LD-API-Version: beta`" + ` header). Beta resources may change or be removed in future versions.

This resource lets you create and manage [release plans](https://launchdarkly.com/docs/home/releases) within a LaunchDarkly environment. A release plan defines how a flag release progresses through ordered phases and steps, which contexts each phase reaches, and which metrics are monitored along the way. Environment release plans replace release policies.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of this resource in the format `project_key/environment_key/key`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:      true,
				Description:   addForceNewDescription("The release plan's project key.", true),
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			ENVIRONMENT_KEY: schema.StringAttribute{
				Required:      true,
				Description:   addForceNewDescription("The key of the environment the release plan belongs to.", true),
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			KEY: schema.StringAttribute{
				Required:      true,
				Description:   addForceNewDescription("The unique human-readable key that references the release plan.", true),
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			NAME: schema.StringAttribute{
				Required:    true,
				Description: "The human-friendly name for the release plan.",
			},
			DESCRIPTION: schema.StringAttribute{
				Optional:    true,
				Description: "The description of the release plan's purpose.",
			},
			RELEASE_METHOD: schema.StringAttribute{
				Computed:      true,
				Description:   "The release method the plan uses. Currently always `progressive_release`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			VERSION: schema.Int64Attribute{
				Computed:    true,
				Description: "The release plan's version. Starts at 1 and increments on every update.",
			},
			IS_DEFAULT: schema.BoolAttribute{
				Computed:      true,
				Description:   "Whether this plan is the environment's default release plan. Manage the default in the LaunchDarkly UI; this resource only reads it.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			PROGRESSIVE_RELEASE: schema.SingleNestedAttribute{
				Required:    true,
				Description: "Configuration for the progressive release this plan runs.",
				Attributes: map[string]schema.Attribute{
					RANDOMIZATION_UNIT: schema.StringAttribute{
						Required:    true,
						Description: "The context kind key used to bucket contexts for allocation.",
						Validators:  []validator.String{keyValidator()},
					},
					ADVANCEMENT_MODE: schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Whether a release started from this plan advances to the next phase automatically once the current phase's steps complete (`automatic`) or waits for a user or API action (`manual`). LaunchDarkly assigns a default when omitted.",
						Validators: []validator.String{
							stringvalidator.OneOf(ADVANCEMENT_MODE_MANUAL, ADVANCEMENT_MODE_AUTOMATIC),
						},
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					CLEANUP_BEHAVIOR: schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "What happens to the flag's targeting rules once a release started from this plan completes. `remove_rules_and_update_default_rule` removes the release's rules and updates the flag's default rule to serve the target variation; `leave_rules_in_place` leaves them untouched. Defaults to `remove_rules_and_update_default_rule`.",
						Validators: []validator.String{
							stringvalidator.OneOf(CLEANUP_BEHAVIOR_REMOVE_RULES, CLEANUP_BEHAVIOR_LEAVE_RULES),
						},
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					PHASES: schema.ListNestedAttribute{
						Required:    true,
						Description: "An ordered list of phases the release progresses through.",
						Validators: []validator.List{
							listvalidator.SizeAtLeast(1),
						},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								DESCRIPTION: schema.StringAttribute{
									Optional:    true,
									Description: "The description of this phase.",
								},
								STEPS: schema.ListNestedAttribute{
									Required:    true,
									Description: "Ordered allocation increments within this phase.",
									Validators: []validator.List{
										listvalidator.SizeAtLeast(1),
									},
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											ALLOCATION: schema.Int64Attribute{
												Required:    true,
												Description: "The traffic allocation for this step, in per-100,000 units. `100000` is 100% of traffic.",
												Validators: []validator.Int64{
													int64validator.Between(0, RELEASE_PLAN_MAX_ALLOCATION),
												},
											},
											DURATION_MILLIS: schema.Int64Attribute{
												Required:    true,
												Description: "How long this step holds, in milliseconds, before the next step starts.",
												Validators: []validator.Int64{
													int64validator.AtLeast(0),
												},
											},
										},
									},
								},
								CLAUSES: environmentReleasePlanClausesSchema(),
							},
						},
					},
					METRICS: schema.MapNestedAttribute{
						Optional:    true,
						Description: "Metrics and metric groups to monitor during the rollout, keyed by metric or metric group key.",
						Validators: []validator.Map{
							mapvalidator.KeysAre(keyValidator()),
						},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								IS_GROUP: schema.BoolAttribute{
									Required:    true,
									Description: "Whether the key identifies a metric group (`true`) or a single metric (`false`).",
								},
								AUTO_ROLLBACK: schema.BoolAttribute{
									Optional:    true,
									Computed:    true,
									Default:     booldefault.StaticBool(false),
									Description: "Whether to automatically roll back the release if this metric or metric group regresses. Defaults to `false`.",
								},
							},
						},
					},
				},
			},
		},
	}
}

func environmentReleasePlanClausesSchema() schema.ListNestedAttribute {
	clauses := frameworkClausesResourceAttribute()
	clauses.Description = "Targeting clauses scoping this phase's audience. Must include the randomization unit clause (`attribute = \"kind\"`, `op = \"in\"`, `values = [<randomization_unit>]`), so the release only reaches contexts of that kind. To release to every context of the randomization unit, provide exactly that clause."
	return clauses
}

func (r *EnvironmentReleasePlanResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *EnvironmentReleasePlanResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg EnvironmentReleasePlanResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateEnvironmentReleasePlanPhaseClauses(ctx, cfg.ProgressiveRelease, &resp.Diagnostics)
}

func (r *EnvironmentReleasePlanResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EnvironmentReleasePlanResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	envKey := plan.EnvironmentKey.ValueString()
	key := plan.Key.ValueString()

	progressive, d := environmentReleasePlanProgressiveReleaseToAPI(ctx, plan.ProgressiveRelease)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	post := environmentReleasePlanRequest{
		Key:                key,
		Name:               plan.Name.ValueString(),
		Description:        stringPointerFromAttr(plan.Description),
		ReleaseMethod:      RELEASE_PLAN_METHOD_PROGRESSIVE,
		ProgressiveRelease: progressive,
	}

	created, err := createEnvironmentReleasePlan(ctx, r.client, projectKey, envKey, post)
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error creating environment release plan %q in environment %q of project %q", key, envKey, projectKey), err)
		return
	}

	environmentReleasePlanToModel(ctx, projectKey, envKey, created, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EnvironmentReleasePlanResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data EnvironmentReleasePlanResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	envKey := data.EnvironmentKey.ValueString()
	key := data.Key.ValueString()

	plan, res, err := getEnvironmentReleasePlan(ctx, r.client, projectKey, envKey, key)
	if err != nil {
		if isStatusNotFound(res) {
			resp.State.RemoveResource(ctx)
			return
		}
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Failed to get environment release plan %q in environment %q of project %q", key, envKey, projectKey), err)
		return
	}

	environmentReleasePlanToModel(ctx, projectKey, envKey, plan, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EnvironmentReleasePlanResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EnvironmentReleasePlanResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	envKey := plan.EnvironmentKey.ValueString()
	key := plan.Key.ValueString()

	progressive, d := environmentReleasePlanProgressiveReleaseToAPI(ctx, plan.ProgressiveRelease)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	put := environmentReleasePlanRequest{
		Name:               plan.Name.ValueString(),
		Description:        stringPointerFromAttr(plan.Description),
		ReleaseMethod:      RELEASE_PLAN_METHOD_PROGRESSIVE,
		ProgressiveRelease: progressive,
	}

	updated, err := updateEnvironmentReleasePlan(ctx, r.client, projectKey, envKey, key, put)
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error updating environment release plan %q in environment %q of project %q", key, envKey, projectKey), err)
		return
	}

	environmentReleasePlanToModel(ctx, projectKey, envKey, updated, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EnvironmentReleasePlanResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data EnvironmentReleasePlanResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key := data.Key.ValueString()
	res, err := deleteEnvironmentReleasePlan(ctx, r.client, data.ProjectKey.ValueString(), data.EnvironmentKey.ValueString(), key)
	if err != nil {
		if isStatusNotFound(res) {
			return
		}
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error deleting environment release plan %q", key), err)
	}
}

func (r *EnvironmentReleasePlanResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectKey, envKey, key, err := environmentReleasePlanIdToKeys(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), projectKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(ENVIRONMENT_KEY), envKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(KEY), key)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
