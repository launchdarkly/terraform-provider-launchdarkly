package launchdarkly

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var (
	_ resource.Resource                     = &ProjectResource{}
	_ resource.ResourceWithImportState      = &ProjectResource{}
	_ resource.ResourceWithConfigValidators = &ProjectResource{}
	_ resource.ResourceWithModifyPlan       = &ProjectResource{}
)

type ProjectResource struct {
	client *Client
}

type ProjectResourceModel struct {
	ID                                   types.String `tfsdk:"id"`
	Key                                  types.String `tfsdk:"key"`
	Name                                 types.String `tfsdk:"name"`
	IncludeInSnippet                     types.Bool   `tfsdk:"include_in_snippet"`
	DefaultClientSideAvailability        types.List   `tfsdk:"default_client_side_availability"`
	Tags                                 types.Set    `tfsdk:"tags"`
	Environments                         types.List   `tfsdk:"environments"`
	RequireViewAssociationForNewFlags    types.Bool   `tfsdk:"require_view_association_for_new_flags"`
	RequireViewAssociationForNewSegments types.Bool   `tfsdk:"require_view_association_for_new_segments"`
}

// projectCSAAttrTypes describes the inner attribute set of the
// default_client_side_availability block.
var projectCSAAttrTypes = map[string]attr.Type{
	USING_ENVIRONMENT_ID: types.BoolType,
	USING_MOBILE_KEY:     types.BoolType,
}

func NewProjectResource() resource.Resource {
	return &ProjectResource{}
}

func (r *ProjectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *ProjectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Provides a LaunchDarkly project resource.

This resource allows you to create and manage projects within your LaunchDarkly organization.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			KEY: schema.StringAttribute{
				Required:      true,
				Description:   addForceNewDescription("The project's unique key.", true),
				Validators:    []validator.String{keyAndLengthValidator(1, 100)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			NAME: schema.StringAttribute{
				Required:    true,
				Description: "The project's name.",
			},
			INCLUDE_IN_SNIPPET: schema.BoolAttribute{
				Optional:           true,
				Computed:           true,
				Description:        "Whether feature flags created under the project should be available to client-side SDKs by default. Please migrate to `default_client_side_availability` to maintain future compatibility.",
				DeprecationMessage: "'include_in_snippet' is now deprecated. Please migrate to 'default_client_side_availability' to maintain future compatibility.",
			},
			TAGS: schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators:  []validator.Set{setvalidator.ValueStringsAre(tagValidator())},
				Description: "Tags associated with your resource.",
			},
			REQUIRE_VIEW_ASSOCIATION_FOR_NEW_FLAGS: schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether new flags created in this project must be associated with at least one view.",
			},
			REQUIRE_VIEW_ASSOCIATION_FOR_NEW_SEGMENTS: schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether new segments created in this project must be associated with at least one view.",
			},
		},
		Blocks: map[string]schema.Block{
			DEFAULT_CLIENT_SIDE_AVAILABILITY: schema.ListNestedBlock{
				Description: "A block describing which client-side SDKs can use new flags by default.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						USING_ENVIRONMENT_ID: schema.BoolAttribute{Required: true},
						USING_MOBILE_KEY:     schema.BoolAttribute{Required: true},
					},
				},
			},
			ENVIRONMENTS: projectEnvironmentsBlock(),
		},
	}
}

func (r *ProjectResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.Conflicting(
			path.MatchRoot(INCLUDE_IN_SNIPPET),
			path.MatchRoot(DEFAULT_CLIENT_SIDE_AVAILABILITY),
		),
	}
}

func (r *ProjectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

// ModifyPlan ports the IIS-side half of customizeProjectDiff: when
// neither IIS nor CSA is declared in config, set include_in_snippet =
// false so terraform sees a stable Computed value matching LD's
// backend default. The CSA half of customizeProjectDiff cannot be
// ported: framework ListNestedBlock cannot be Computed at the block
// level, so emitting a block when config has zero blocks fails
// terraform-core's "plan count must match config count" gate (Phase 3
// gotcha #3). The block-shape parity is preserved by Read mirroring
// the prior state's block presence (see readIntoModel + the CSA
// helpers below).
func (r *ProjectResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	if r.client == nil {
		return
	}
	var config ProjectResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	iisNotSet := config.IncludeInSnippet.IsNull() || config.IncludeInSnippet.IsUnknown()
	csaEmpty := config.DefaultClientSideAvailability.IsNull() ||
		config.DefaultClientSideAvailability.IsUnknown() ||
		len(config.DefaultClientSideAvailability.Elements()) == 0
	if !(iisNotSet && csaEmpty) {
		return
	}

	var plan ProjectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.IncludeInSnippet = types.BoolValue(false)
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

// projectCSAValueFromAPI emits the CSA block matching the prior state
// shape: if prior was empty (or null), keep empty so terraform doesn't
// see a count-1 state for a count-0 plan; if prior was populated, emit
// the API's current values. This is the framework analogue of SDKv2's
// Optional+Computed TypeList for blocks — framework blocks can't be
// Computed at the block level.
func projectCSAValueFromAPI(ctx context.Context, csa *ldapi.ClientSideAvailability, prior basetypes.ListValue) (basetypes.ListValue, diag.Diagnostics) {
	objectType := types.ObjectType{AttrTypes: projectCSAAttrTypes}
	priorEmpty := prior.IsNull() || prior.IsUnknown() || len(prior.Elements()) == 0
	if priorEmpty || csa == nil {
		return types.ListValue(objectType, []attr.Value{})
	}
	usingEnv := false
	if csa.UsingEnvironmentId != nil {
		usingEnv = *csa.UsingEnvironmentId
	}
	usingMobile := false
	if csa.UsingMobileKey != nil {
		usingMobile = *csa.UsingMobileKey
	}
	obj, diags := types.ObjectValue(projectCSAAttrTypes, map[string]attr.Value{
		USING_ENVIRONMENT_ID: types.BoolValue(usingEnv),
		USING_MOBILE_KEY:     types.BoolValue(usingMobile),
	})
	list, d := types.ListValue(objectType, []attr.Value{obj})
	diags.Append(d...)
	return list, diags
}

func (r *ProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProjectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectKey := plan.Key.ValueString()

	envPosts, diags := environmentPostsFromPlan(ctx, plan.Environments)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := ldapi.ProjectPost{
		Name: plan.Name.ValueString(),
		Key:  projectKey,
	}
	if len(envPosts) > 0 {
		body.Environments = envPosts
	}

	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.ProjectsApi.PostProject(r.client.ctx).ProjectPost(body).Execute()
		return e
	})
	if err != nil {
		if !isTimeoutError(err) {
			resp.Diagnostics.AddError(fmt.Sprintf("failed to create project with name %s and projectKey %s: %s", plan.Name.ValueString(), projectKey, handleLdapiErr(err).Error()), "")
			return
		}
		log.Printf("[DEBUG] Network timeout when creating project %q. This can happen with 20+ environments. Apply usually still succeeds.\n", projectKey)
	}

	if d := r.applyProjectUpdates(ctx, projectKey, plan, ProjectResourceModel{}, true); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}
	r.readIntoModel(ctx, projectKey, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = types.StringValue(projectKey)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProjectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readIntoModel(ctx, data.Key.ValueString(), &data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ProjectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectKey := plan.Key.ValueString()

	if d := r.applyProjectUpdates(ctx, projectKey, plan, state, false); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}
	r.readIntoModel(ctx, projectKey, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = types.StringValue(projectKey)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ProjectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, e := r.client.ld.ProjectsApi.DeleteProject(r.client.ctx, data.Key.ValueString()).Execute()
		return e
	})
	if err != nil && !isTimeoutError(err) {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to delete project with key %q: %s", data.Key.ValueString(), handleLdapiErr(err).Error()), "")
	}
}

func (r *ProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(KEY), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// applyProjectUpdates issues all the patch + nested-environment +
// view-association calls SDKv2 made after PostProject. Used by both
// Create (state empty) and Update paths.
func (r *ProjectResource) applyProjectUpdates(ctx context.Context, projectKey string, plan, state ProjectResourceModel, isCreate bool) diag.Diagnostics {
	var diags diag.Diagnostics

	tags, d := stringSliceFromSet(ctx, plan.Tags)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	patches := []ldapi.PatchOperation{
		patchReplace("/name", plan.Name.ValueString()),
		patchReplace("/tags", &tags),
	}

	csaPlanned := !plan.DefaultClientSideAvailability.IsNull() && !plan.DefaultClientSideAvailability.IsUnknown() && len(plan.DefaultClientSideAvailability.Elements()) > 0
	csaChanged := isCreate || !plan.DefaultClientSideAvailability.Equal(state.DefaultClientSideAvailability)
	iisChanged := isCreate || !plan.IncludeInSnippet.Equal(state.IncludeInSnippet)

	switch {
	case csaPlanned && csaChanged:
		csa, d := csaPostFromList(ctx, plan.DefaultClientSideAvailability)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
		patches = append(patches, patchReplace("/defaultClientSideAvailability", csa))
	case !plan.IncludeInSnippet.IsNull() && !plan.IncludeInSnippet.IsUnknown() && iisChanged:
		patches = append(patches, patchReplace("/defaultClientSideAvailability", &ldapi.ClientSideAvailabilityPost{
			UsingEnvironmentId: plan.IncludeInSnippet.ValueBool(),
			UsingMobileKey:     true,
		}))
	}

	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.ProjectsApi.PatchProject(r.client.ctx, projectKey).PatchOperation(patches).Execute()
		return e
	})
	if err != nil {
		diags.AddError(fmt.Sprintf("failed to update project with key %q: %s", projectKey, handleLdapiErr(err).Error()), "")
		return diags
	}

	// View association settings — handled separately via raw HTTP since not in the official API client.
	flagsChanged := isCreate || !plan.RequireViewAssociationForNewFlags.Equal(state.RequireViewAssociationForNewFlags)
	segmentsChanged := isCreate || !plan.RequireViewAssociationForNewSegments.Equal(state.RequireViewAssociationForNewSegments)
	if flagsChanged || segmentsChanged {
		if err := patchProjectViewSettings(ctx, r.client, projectKey,
			plan.RequireViewAssociationForNewFlags.ValueBool(),
			plan.RequireViewAssociationForNewSegments.ValueBool(),
			flagsChanged, segmentsChanged); err != nil {
			diags.AddError(fmt.Sprintf("failed to update view association settings for project %q: %s", projectKey, err.Error()), "")
			return diags
		}
	}

	// Environment reconciliation
	planEnvs, d := environmentModelsFromList(ctx, plan.Environments)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	stateEnvs := map[string]environmentBlockModel{}
	if !isCreate {
		stateEnvList, d := environmentModelsFromList(ctx, state.Environments)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
		for _, e := range stateEnvList {
			stateEnvs[e.Key.ValueString()] = e
		}
	}

	project, _, err := getFullProject(r.client, projectKey)
	if err != nil {
		diags.AddError(fmt.Sprintf("failed to load project %q before updating environments: %s", projectKey, handleLdapiErr(err).Error()), "")
		return diags
	}

	desired := map[string]bool{}
	for _, env := range planEnvs {
		envKey := env.Key.ValueString()
		desired[envKey] = true
		if !environmentExistsInProject(*project, envKey) {
			envPost, d := environmentPostFromModel(ctx, env)
			diags.Append(d...)
			if diags.HasError() {
				return diags
			}
			err := r.client.withConcurrency(r.client.ctx, func() error {
				_, _, e := r.client.ld.EnvironmentsApi.PostEnvironment(r.client.ctx, projectKey).EnvironmentPost(envPost).Execute()
				return e
			})
			if err != nil {
				diags.AddError(fmt.Sprintf("failed to create environment %q in project %q: %s", envKey, projectKey, handleLdapiErr(err).Error()), "")
				return diags
			}
		}
		oldEnv, hadOld := stateEnvs[envKey]
		envPatch, d := environmentPatchFromModels(ctx, oldEnv, hadOld, env)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
		err := r.client.withConcurrency(r.client.ctx, func() error {
			_, _, e := r.client.ld.EnvironmentsApi.PatchEnvironment(r.client.ctx, projectKey, envKey).PatchOperation(envPatch).Execute()
			return e
		})
		if err != nil {
			diags.AddError(fmt.Sprintf("failed to update environment %q in project %q: %s", envKey, projectKey, handleLdapiErr(err).Error()), "")
			return diags
		}
	}
	// Delete environments removed from config
	for envKey := range stateEnvs {
		if desired[envKey] {
			continue
		}
		err := r.client.withConcurrency(r.client.ctx, func() error {
			_, e := r.client.ld.EnvironmentsApi.DeleteEnvironment(r.client.ctx, projectKey, envKey).Execute()
			return e
		})
		if err != nil {
			diags.AddError(fmt.Sprintf("failed to delete environment %q in project %q: %s", envKey, projectKey, handleLdapiErr(err).Error()), "")
			return diags
		}
	}
	return diags
}

func (r *ProjectResource) readIntoModel(ctx context.Context, projectKey string, data *ProjectResourceModel, diags *diag.Diagnostics) {
	project, res, err := getFullProject(r.client, projectKey)
	if isStatusNotFound(res) {
		data.ID = types.StringNull()
		return
	}
	if err != nil {
		diags.AddError(fmt.Sprintf("failed to get project with key %q: %s", projectKey, err.Error()), "")
		return
	}

	data.ID = types.StringValue(projectKey)
	data.Key = types.StringValue(project.Key)
	data.Name = types.StringValue(project.Name)

	tagsSet, d := setFromStringSlice(ctx, project.Tags)
	diags.Append(d...)
	data.Tags = tagsSet

	csaList, d := projectCSAValueFromAPI(ctx, project.DefaultClientSideAvailability, data.DefaultClientSideAvailability)
	diags.Append(d...)
	data.DefaultClientSideAvailability = csaList

	// include_in_snippet mirrors the deprecated API field; LD's
	// IncludeInSnippetByDefault is the canonical source.
	data.IncludeInSnippet = types.BoolValue(project.IncludeInSnippetByDefault)

	// Environments — preserve config order, then append unmanaged ones.
	envItems := []ldapi.Environment{}
	if project.Environments != nil {
		envItems = project.Environments.Items
	}
	envList, d := environmentsListFromAPI(ctx, envItems, data.Environments)
	diags.Append(d...)
	data.Environments = envList

	// View association settings
	settings, err := getProjectViewSettings(ctx, r.client, projectKey)
	if err != nil {
		log.Printf("[WARN] failed to get view association settings for project %q: %v", projectKey, err)
		// Keep prior state for these fields; nothing to set if unknown.
		if data.RequireViewAssociationForNewFlags.IsNull() || data.RequireViewAssociationForNewFlags.IsUnknown() {
			data.RequireViewAssociationForNewFlags = types.BoolValue(false)
		}
		if data.RequireViewAssociationForNewSegments.IsNull() || data.RequireViewAssociationForNewSegments.IsUnknown() {
			data.RequireViewAssociationForNewSegments = types.BoolValue(false)
		}
		return
	}
	data.RequireViewAssociationForNewFlags = types.BoolValue(settings.RequireViewAssociationForNewFlags)
	data.RequireViewAssociationForNewSegments = types.BoolValue(settings.RequireViewAssociationForNewSegments)
}

// csaPostFromList extracts UsingEnvironmentId / UsingMobileKey from a
// framework single-element ListValue.
func csaPostFromList(ctx context.Context, l types.List) (*ldapi.ClientSideAvailabilityPost, diag.Diagnostics) {
	if l.IsNull() || l.IsUnknown() || len(l.Elements()) == 0 {
		return nil, nil
	}
	type csaModel struct {
		UsingEnvironmentId types.Bool `tfsdk:"using_environment_id"`
		UsingMobileKey     types.Bool `tfsdk:"using_mobile_key"`
	}
	var models []csaModel
	diags := l.ElementsAs(ctx, &models, false)
	if diags.HasError() {
		return nil, diags
	}
	if len(models) == 0 {
		return nil, diags
	}
	return &ldapi.ClientSideAvailabilityPost{
		UsingEnvironmentId: models[0].UsingEnvironmentId.ValueBool(),
		UsingMobileKey:     models[0].UsingMobileKey.ValueBool(),
	}, diags
}
