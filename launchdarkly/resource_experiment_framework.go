package launchdarkly

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var (
	_ resource.Resource                = &ExperimentResource{}
	_ resource.ResourceWithImportState = &ExperimentResource{}
)

type ExperimentResource struct {
	client *Client
}

type ExperimentResourceModel struct {
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
	Iteration      types.Object `tfsdk:"iteration"`
}

func NewExperimentResource() resource.Resource { return &ExperimentResource{} }

func (r *ExperimentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_experiment"
}

func (r *ExperimentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly experiment resource.\n\nThis resource lets you create and manage experiments. An experiment is created together with its first iteration. The iteration is created as a draft; starting and stopping iterations is a runtime action that is not managed by Terraform. Changing any `iteration` field creates a new draft iteration.\n\nThe LaunchDarkly API does not support deleting experiments, so destroying this resource archives the experiment instead. The `project_key`, `environment_key`, `key`, `maintainer_id`, `holdout_id`, and `tags` fields cannot be updated in place; a change forces the experiment to be archived and a new one created.\n\nTo learn more, read [Creating experiments](https://launchdarkly.com/docs/home/experimentation/create).",
		Attributes: map[string]schema.Attribute{
			ID: schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of this resource, in the format `project_key/environment_key/experiment_key`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:      true,
				Description:   "The project key. A change in this field will force the destruction of the existing resource and the creation of a new one.",
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			ENVIRONMENT_KEY: schema.StringAttribute{
				Required:      true,
				Description:   "The environment key. A change in this field will force the destruction of the existing resource and the creation of a new one.",
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			KEY: schema.StringAttribute{
				Required:      true,
				Description:   "The unique key that references the experiment. A change in this field will force the destruction of the existing resource and the creation of a new one.",
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			NAME: schema.StringAttribute{
				Required:    true,
				Description: "The human-friendly name of the experiment.",
			},
			DESCRIPTION: schema.StringAttribute{
				Optional:    true,
				Description: "A description of the experiment.",
			},
			MAINTAINER_ID: schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the member who maintains the experiment. If not set, the member associated with the API token is used. A change in this field will force the destruction of the existing resource and the creation of a new one.",
				Validators:    []validator.String{idValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			HOLDOUT_ID: schema.StringAttribute{
				Optional:      true,
				Description:   "The ID of the holdout to associate with this experiment. A change in this field will force the destruction of the existing resource and the creation of a new one.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			TAGS: schema.SetAttribute{
				Optional:      true,
				ElementType:   types.StringType,
				Description:   "Tags associated with the experiment. A change in this field will force the destruction of the existing resource and the creation of a new one.",
				Validators:    []validator.Set{setvalidator.ValueStringsAre(tagValidator())},
				PlanModifiers: []planmodifier.Set{setplanmodifier.RequiresReplace()},
			},
			ARCHIVED: schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the experiment is archived. Set to `true` to archive the experiment, or `false` to restore an archived experiment. Defaults to `false`.",
			},
			ITERATION: schema.SingleNestedAttribute{
				Required:    true,
				Description: "The configuration of the experiment's iteration. Changing any value creates a new draft iteration.",
				Attributes:  experimentIterationSchema(),
			},
		},
	}
}

func experimentIterationSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		HYPOTHESIS: schema.StringAttribute{
			Required:    true,
			Description: "The expected outcome of this experiment.",
		},
		CAN_RESHUFFLE_TRAFFIC: schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(true),
			Description: "Whether to allow the experiment to reassign traffic to different variations when you change the traffic allocation. Defaults to `true`.",
		},
		RANDOMIZATION_UNIT: schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "The unit of randomization for this iteration. Must match the key of a context kind enabled for experiments in the project. Defaults to the project's default randomization unit.",
		},
		ATTRIBUTES: schema.SetAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: "The context attributes that this iteration's results can be sliced by.",
		},
		PRIMARY_SINGLE_METRIC_KEY: schema.StringAttribute{
			Optional:    true,
			Description: "The key of the primary metric for this experiment. Either `primary_single_metric_key` or `primary_funnel_key` must be set.",
		},
		PRIMARY_FUNNEL_KEY: schema.StringAttribute{
			Optional:    true,
			Description: "The key of the primary funnel metric group for this experiment. Either `primary_single_metric_key` or `primary_funnel_key` must be set.",
		},
		METRICS: schema.ListNestedAttribute{
			Required:    true,
			Description: "The metrics measured by this experiment.",
			Validators:  []validator.List{listvalidator.SizeAtLeast(1)},
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					KEY: schema.StringAttribute{
						Required:    true,
						Description: "The metric or metric group key.",
					},
					IS_GROUP: schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "Whether this references a metric group (`true`) or a single metric (`false`). Defaults to `false`.",
					},
				},
			},
		},
		TREATMENTS: schema.ListNestedAttribute{
			Required:    true,
			Description: "The treatments (variations) being compared in this experiment. At least two treatments are required.",
			Validators:  []validator.List{listvalidator.SizeAtLeast(2)},
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					NAME: schema.StringAttribute{
						Required:    true,
						Description: "The treatment name.",
					},
					BASELINE: schema.BoolAttribute{
						Required:    true,
						Description: "Whether this treatment is the baseline to compare other treatments against.",
					},
					ALLOCATION_PERCENT: schema.StringAttribute{
						Required:    true,
						Description: "The percentage of experiment traffic allocated to this treatment, as a string (for example, `\"50\"`).",
					},
					PARAMETERS: schema.ListNestedAttribute{
						Required:    true,
						Description: "The flag and variation served for this treatment.",
						Validators:  []validator.List{listvalidator.SizeAtLeast(1)},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								FLAG_KEY: schema.StringAttribute{
									Required:    true,
									Description: "The flag key.",
								},
								VARIATION_ID: schema.StringAttribute{
									Required:    true,
									Description: "The ID of the flag variation to serve for this treatment.",
								},
							},
						},
					},
				},
			},
		},
		FLAGS: schema.MapNestedAttribute{
			Required:    true,
			Description: "The flags used in this experiment, keyed by flag key.",
			Validators:  []validator.Map{mapvalidator.SizeAtLeast(1)},
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					RULE_ID: schema.StringAttribute{
						Required:    true,
						Description: "The ID of the variation or rollout of the flag to use. Use `fallthrough` for the flag's default targeting behavior when it is on.",
					},
					FLAG_CONFIG_VERSION: schema.Int64Attribute{
						Required:    true,
						Description: "The flag configuration version to pin the experiment to.",
					},
					NOT_IN_EXPERIMENT_VARIATION_ID: schema.StringAttribute{
						Optional:    true,
						Description: "The ID of the variation to serve to traffic that is not part of the experiment analysis. Defaults to the baseline treatment's variation.",
					},
				},
			},
		},
	}
}

func (r *ExperimentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *ExperimentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ExperimentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	iterInput := iterationInputFromObject(ctx, plan.Iteration, &resp.Diagnostics)
	tags, d := stringSliceFromSet(ctx, plan.Tags)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	environmentKey := plan.EnvironmentKey.ValueString()
	key := plan.Key.ValueString()

	post := ldapi.ExperimentPost{
		Name:      plan.Name.ValueString(),
		Key:       key,
		Iteration: iterInput,
	}
	if v := plan.Description.ValueString(); v != "" {
		post.Description = &v
	}
	if !plan.MaintainerID.IsNull() && !plan.MaintainerID.IsUnknown() {
		if v := plan.MaintainerID.ValueString(); v != "" {
			post.MaintainerId = &v
		}
	}
	if v := plan.HoldoutID.ValueString(); v != "" {
		post.HoldoutId = &v
	}
	if len(tags) > 0 {
		post.Tags = tags
	}

	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.ExperimentsApi.CreateExperiment(r.client.ctx, projectKey, environmentKey).ExperimentPost(post).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Failed to create experiment %q", key), err)
		return
	}

	// archived defaults to false on create; an explicit true is applied as a
	// follow-up archive instruction.
	if plan.Archived.ValueBool() {
		if err := r.patchInstructions(projectKey, environmentKey, key, []map[string]interface{}{{KIND: "archiveExperiment"}}); err != nil {
			addLdapiError(&resp.Diagnostics, "Failed to archive newly created experiment", err)
			return
		}
	}

	plan.ID = types.StringValue(experimentID(projectKey, environmentKey, key))
	iteration := plan.Iteration
	r.readIntoModel(ctx, projectKey, environmentKey, key, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	// Preserve the configured iteration; the API response shape cannot be
	// converted back into the iteration input faithfully.
	plan.Iteration = iteration
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ExperimentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ExperimentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readIntoModel(ctx, data.ProjectKey.ValueString(), data.EnvironmentKey.ValueString(), data.Key.ValueString(), &data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ExperimentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ExperimentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	environmentKey := plan.EnvironmentKey.ValueString()
	key := plan.Key.ValueString()

	var instructions []map[string]interface{}
	if !plan.Name.Equal(state.Name) {
		instructions = append(instructions, map[string]interface{}{KIND: "updateName", VALUE: plan.Name.ValueString()})
	}
	if !plan.Description.Equal(state.Description) {
		instructions = append(instructions, map[string]interface{}{KIND: "updateDescription", VALUE: plan.Description.ValueString()})
	}
	if !plan.Archived.Equal(state.Archived) {
		if plan.Archived.ValueBool() {
			instructions = append(instructions, map[string]interface{}{KIND: "archiveExperiment"})
		} else {
			instructions = append(instructions, map[string]interface{}{KIND: "restoreExperiment"})
		}
	}

	if len(instructions) > 0 {
		if err := r.patchInstructions(projectKey, environmentKey, key, instructions); err != nil {
			addLdapiError(&resp.Diagnostics, fmt.Sprintf("Failed to update experiment %q", key), err)
			return
		}
	}

	// A change to the iteration configuration creates a new draft iteration.
	if !plan.Iteration.Equal(state.Iteration) {
		iterInput := iterationInputFromObject(ctx, plan.Iteration, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		err := r.client.withConcurrency(r.client.ctx, func() error {
			_, _, e := r.client.ld.ExperimentsApi.CreateIteration(r.client.ctx, projectKey, environmentKey, key).IterationInput(iterInput).Execute()
			return e
		})
		if err != nil {
			addLdapiError(&resp.Diagnostics, fmt.Sprintf("Failed to create new iteration for experiment %q", key), err)
			return
		}
	}

	iteration := plan.Iteration
	r.readIntoModel(ctx, projectKey, environmentKey, key, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Iteration = iteration
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete archives the experiment. The LaunchDarkly API has no endpoint to
// delete an experiment, so destroying the resource archives it and removes it
// from Terraform state.
func (r *ExperimentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ExperimentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if data.Archived.ValueBool() {
		// Already archived; nothing to do.
		return
	}
	err := r.patchInstructions(data.ProjectKey.ValueString(), data.EnvironmentKey.ValueString(), data.Key.ValueString(), []map[string]interface{}{{KIND: "archiveExperiment"}})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Failed to archive experiment %q on destroy", data.Key.ValueString()), err)
	}
}

func (r *ExperimentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectKey, environmentKey, key, err := experimentIDToKeys(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), projectKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(ENVIRONMENT_KEY), environmentKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(KEY), key)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(ID), req.ID)...)
}

func (r *ExperimentResource) patchInstructions(projectKey, environmentKey, key string, instructions []map[string]interface{}) error {
	patch := ldapi.ExperimentPatchInput{Instructions: instructions}
	return r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.ExperimentsApi.PatchExperiment(r.client.ctx, projectKey, environmentKey, key).ExperimentPatchInput(patch).Execute()
		return e
	})
}

func (r *ExperimentResource) readIntoModel(
	ctx context.Context,
	projectKey, environmentKey, key string,
	data *ExperimentResourceModel,
	diags *diag.Diagnostics,
) {
	var experiment *ldapi.Experiment
	var res *http.Response
	err := r.client.withConcurrency(r.client.ctx, func() error {
		var e error
		experiment, res, e = r.client.ld.ExperimentsApi.GetExperiment(r.client.ctx, projectKey, environmentKey, key).Execute()
		return e
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError(fmt.Sprintf("Failed to get experiment %q", key), handleLdapiErr(err).Error())
		return
	}

	data.ID = types.StringValue(experimentID(projectKey, environmentKey, key))
	data.ProjectKey = types.StringValue(projectKey)
	data.EnvironmentKey = types.StringValue(environmentKey)
	data.Key = types.StringValue(experiment.Key)
	data.Name = types.StringValue(experiment.Name)
	data.Description = stringValueOrNullFromPointer(experiment.Description)
	data.MaintainerID = types.StringValue(experiment.MaintainerId)
	data.HoldoutID = stringValueOrNullFromPointer(experiment.HoldoutId)
	data.Archived = types.BoolValue(experiment.ArchivedDate != nil)

	tags, d := setFromStringSliceOrNull(ctx, experiment.Tags)
	diags.Append(d...)
	data.Tags = tags
}
