package launchdarkly

import (
	"context"
	"fmt"
	"net/http"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var (
	_ resource.Resource                = &ReleasePipelineResource{}
	_ resource.ResourceWithImportState = &ReleasePipelineResource{}
	_ resource.ResourceWithModifyPlan  = &ReleasePipelineResource{}
)

type ReleasePipelineResource struct {
	client *Client
	beta   *Client
}

type ReleasePipelineResourceModel struct {
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

func NewReleasePipelineResource() resource.Resource { return &ReleasePipelineResource{} }

func (r *ReleasePipelineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_release_pipeline"
}

func (r *ReleasePipelineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Provides a LaunchDarkly release pipeline resource.

~> **Beta:** This resource wraps a beta LaunchDarkly API. Beta resources may change or be removed in future versions of the provider, and the underlying API may change in backwards-incompatible ways.

This resource allows you to create and manage release pipelines within a LaunchDarkly project. A release pipeline is an ordered set of ` + "`phases`" + `, each phase being a logical grouping of one or more ` + "`audiences`" + ` (LaunchDarkly environments) that flags progress through as they are rolled out. To learn more, read [Release pipelines](https://docs.launchdarkly.com/home/releases/release-pipelines).`,
		Attributes: releasePipelineSchemaAttributes(),
	}
}

func releasePipelineSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:      true,
			Description:   "The ID of this resource in the format `project_key/key`.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		PROJECT_KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The release pipeline's project key.", true),
			Validators:    []validator.String{keyValidator()},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The unique key that references the release pipeline.", true),
			Validators:    []validator.String{keyValidator()},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		NAME: schema.StringAttribute{
			Required:    true,
			Description: "A human-friendly name for the release pipeline.",
		},
		DESCRIPTION: schema.StringAttribute{
			Optional:    true,
			Description: "A description of the release pipeline's purpose.",
		},
		TAGS: schema.SetAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: "Tags associated with the release pipeline.",
			Validators: []validator.Set{
				setvalidator.ValueStringsAre(tagValidator()),
			},
		},
		IS_PROJECT_DEFAULT: schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: addForceNewDescription("Whether this release pipeline is the default pipeline for its project. Can only be set at creation time; the update API does not support changing it.", true),
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
				boolplanmodifier.RequiresReplace(),
			},
		},
		CREATED_AT: schema.StringAttribute{
			Computed:      true,
			Description:   "The release pipeline's creation date represented as a UNIX-style timestamp, in milliseconds since the epoch.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		VERSION: schema.Int64Attribute{
			Computed:    true,
			Description: "The version of the release pipeline.",
		},
		PHASES: schema.ListNestedAttribute{
			Required:    true,
			Description: "An ordered list of the release pipeline's phases. Each phase is a logical grouping of one or more audiences that share attributes for rolling out changes. Must contain at least one phase. The order is significant — flags progress through the phases in order.",
			Validators: []validator.List{
				listvalidator.SizeAtLeast(1),
			},
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					NAME: schema.StringAttribute{
						Required:    true,
						Description: "The release phase name.",
					},
					AUDIENCES: schema.ListNestedAttribute{
						Required:    true,
						Description: "An ordered list of the audiences for this phase. Each audience corresponds to a LaunchDarkly environment. Must contain at least one audience.",
						Validators: []validator.List{
							listvalidator.SizeAtLeast(1),
						},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								ENVIRONMENT_KEY: schema.StringAttribute{
									Required:    true,
									Description: "The key of the LaunchDarkly environment this audience targets.",
									Validators:  []validator.String{keyValidator()},
								},
								NAME: schema.StringAttribute{
									Required:    true,
									Description: "The audience name.",
								},
								SEGMENT_KEYS: schema.SetAttribute{
									Optional:    true,
									ElementType: types.StringType,
									Description: "Segment keys targeted by this audience. Omit the attribute when the audience targets no segments; an explicit empty set is not supported.",
									Validators: []validator.Set{
										setvalidator.ValueStringsAre(keyValidator()),
									},
								},
								CONFIGURATION: schema.SingleNestedAttribute{
									Optional:    true,
									Description: "Release strategy and approval configuration for this audience.",
									Attributes: map[string]schema.Attribute{
										RELEASE_STRATEGY: schema.StringAttribute{
											Required:    true,
											Description: "The release strategy for this audience.",
										},
										REQUIRE_APPROVAL: schema.BoolAttribute{
											Optional:    true,
											Computed:    true,
											Default:     booldefault.StaticBool(false),
											Description: "Whether this audience requires approval before changes are rolled out. Defaults to `false`.",
										},
										NOTIFY_MEMBER_IDS: schema.SetAttribute{
											Optional:    true,
											ElementType: types.StringType,
											Description: "Member IDs notified to review the approval request. Only meaningful when `require_approval` is `true`.",
											Validators: []validator.Set{
												setvalidator.ValueStringsAre(idValidator()),
											},
										},
										NOTIFY_TEAM_KEYS: schema.SetAttribute{
											Optional:    true,
											ElementType: types.StringType,
											Description: "Team keys whose members are notified to review the approval request. Only meaningful when `require_approval` is `true`.",
											Validators: []validator.Set{
												setvalidator.ValueStringsAre(keyValidator()),
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

func (r *ReleasePipelineResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
	if r.client == nil {
		return
	}
	beta, err := newReleasePipelineBetaClient(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build LaunchDarkly beta client", err.Error())
		return
	}
	r.beta = beta
}

func (r *ReleasePipelineResource) betaClient() (*Client, error) {
	if r.beta != nil {
		return r.beta, nil
	}
	return newReleasePipelineBetaClient(r.client)
}

// ModifyPlan marks the computed `version` as unknown whenever a
// user-controlled attribute changes, so the post-apply refresh does not trip
// "inconsistent result after apply".
func (r *ReleasePipelineResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}
	var plan, state ReleasePipelineResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	planVersion := plan.Version
	plan.Version = state.Version
	if !reflect.DeepEqual(plan, state) {
		plan.Version = types.Int64Unknown()
	} else {
		plan.Version = planVersion
	}
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r *ReleasePipelineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ReleasePipelineResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	if exists, err := projectExists(projectKey, r.client); !exists {
		if err != nil {
			resp.Diagnostics.AddError("Failed to check project", err.Error())
			return
		}
		resp.Diagnostics.AddError("Project not found", fmt.Sprintf("cannot find project with key %q", projectKey))
		return
	}

	beta, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}

	var phaseModels []releasePipelinePhaseModel
	resp.Diagnostics.Append(plan.Phases.ElementsAs(ctx, &phaseModels, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	phases, d := releasePipelinePhaseInputsFromModels(ctx, phaseModels)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	key := plan.Key.ValueString()
	name := plan.Name.ValueString()
	input := ldapi.CreateReleasePipelineInput{
		Key:    key,
		Name:   name,
		Phases: phases,
	}
	if description := plan.Description.ValueString(); description != "" {
		input.Description = &description
	}
	tags, d := stringSliceFromSet(ctx, plan.Tags)
	resp.Diagnostics.Append(d...)
	if len(tags) > 0 {
		input.Tags = tags
	}
	if !plan.IsProjectDefault.IsNull() && !plan.IsProjectDefault.IsUnknown() {
		isDefault := plan.IsProjectDefault.ValueBool()
		input.IsProjectDefault = &isDefault
	}

	err = beta.withConcurrency(beta.ctx, func() error {
		_, _, e := beta.ld.ReleasePipelinesBetaApi.PostReleasePipeline(beta.ctx, projectKey).CreateReleasePipelineInput(input).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error creating release pipeline resource: %q", key), err)
		return
	}

	plan.ID = types.StringValue(projectKey + "/" + key)
	r.readIntoModel(ctx, projectKey, key, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ReleasePipelineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ReleasePipelineResourceModel
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

func (r *ReleasePipelineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ReleasePipelineResourceModel
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

	projectKey := plan.ProjectKey.ValueString()
	key := plan.Key.ValueString()

	var phaseModels []releasePipelinePhaseModel
	resp.Diagnostics.Append(plan.Phases.ElementsAs(ctx, &phaseModels, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	phases, d := releasePipelinePhaseInputsFromModels(ctx, phaseModels)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The release pipeline API has no PATCH; updates are a full PUT
	// replacement built from the plan.
	input := ldapi.UpdateReleasePipelineInput{
		Name:   plan.Name.ValueString(),
		Phases: phases,
	}
	if description := plan.Description.ValueString(); description != "" {
		input.Description = &description
	}
	tags, d := stringSliceFromSet(ctx, plan.Tags)
	resp.Diagnostics.Append(d...)
	// Always send tags so removing the last tag clears them server-side.
	input.Tags = tags

	err = beta.withConcurrency(beta.ctx, func() error {
		_, _, e := beta.ld.ReleasePipelinesBetaApi.PutReleasePipeline(beta.ctx, projectKey, key).UpdateReleasePipelineInput(input).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error updating release pipeline resource %q in project %q", key, projectKey), err)
		return
	}

	r.readIntoModel(ctx, projectKey, key, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ReleasePipelineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ReleasePipelineResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	beta, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}
	var res *http.Response
	err = beta.withConcurrency(beta.ctx, func() error {
		var e error
		res, e = beta.ld.ReleasePipelinesBetaApi.DeleteReleasePipeline(beta.ctx, data.ProjectKey.ValueString(), data.Key.ValueString()).Execute()
		return e
	})
	if err != nil {
		if isStatusNotFound(res) {
			return
		}
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error deleting release pipeline resource %q", data.Key.ValueString()), err)
	}
}

func (r *ReleasePipelineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectKey, key, err := releasePipelineIdToKeys(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), projectKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(KEY), key)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *ReleasePipelineResource) readIntoModel(
	ctx context.Context,
	projectKey, key string,
	data *ReleasePipelineResourceModel,
	diags *diag.Diagnostics,
) {
	beta, err := r.betaClient()
	if err != nil {
		diags.AddError("Failed to build beta client", err.Error())
		return
	}

	var pipeline *ldapi.ReleasePipeline
	var res *http.Response
	err = beta.withConcurrency(beta.ctx, func() error {
		pipeline, res, err = beta.ld.ReleasePipelinesBetaApi.GetReleasePipelineByKey(beta.ctx, projectKey, key).Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError(fmt.Sprintf("Failed to get release pipeline %q", key), handleLdapiErr(err).Error())
		return
	}

	data.ID = types.StringValue(projectKey + "/" + key)
	data.ProjectKey = types.StringValue(projectKey)
	data.Key = types.StringValue(pipeline.Key)
	data.Name = types.StringValue(pipeline.Name)
	// Optional-only attr: null-when-empty for plan-apply consistency.
	data.Description = stringValueOrNullFromPointer(pipeline.Description)
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

	// Optional-only Set attr: preserve the config's null-vs-empty intent.
	tagsSet, d := setFromStringSlicePreservingPlan(ctx, pipeline.Tags, data.Tags)
	diags.Append(d...)
	data.Tags = tagsSet

	phasesList, d := releasePipelinePhasesToList(ctx, pipeline.Phases, data.Phases)
	diags.Append(d...)
	data.Phases = phasesList
}
