package launchdarkly

import (
	"context"
	"fmt"
	"log"

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
	_ resource.Resource                 = &ProjectResource{}
	_ resource.ResourceWithImportState  = &ProjectResource{}
	_ resource.ResourceWithModifyPlan   = &ProjectResource{}
	_ resource.ResourceWithUpgradeState = &ProjectResource{}
)

type ProjectResource struct {
	client *Client
}

type ProjectResourceModel struct {
	ID                                   types.String `tfsdk:"id"`
	Key                                  types.String `tfsdk:"key"`
	Name                                 types.String `tfsdk:"name"`
	DefaultClientSideAvailability        types.Object `tfsdk:"default_client_side_availability"`
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
		Version: 1,
		Description: `Provides a LaunchDarkly project resource.

This resource allows you to create and manage projects within your LaunchDarkly organization.`,
		Attributes: projectSchemaAttributes(),
	}
}

func projectSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
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
		TAGS: schema.SetAttribute{
			Optional:    true,
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
		DEFAULT_CLIENT_SIDE_AVAILABILITY: schema.SingleNestedAttribute{
			Optional:    true,
			Description: "Which client-side SDKs can use new flags by default.",
			Attributes: map[string]schema.Attribute{
				USING_ENVIRONMENT_ID: schema.BoolAttribute{Required: true},
				USING_MOBILE_KEY:     schema.BoolAttribute{Required: true},
			},
		},
		ENVIRONMENTS: projectEnvironmentsAttribute(),
	}
}

// approvalSettingsMatchesAPIDefaults reports true if an
// approval_settings item carries the field values the LD API auto-fills
// for an environment whose approvals have never been configured. The
// v2.29 SDKv2 provider persisted these defaults in state even when the
// user never wrote them, so the upgrader uses this heuristic to drop
// them in favour of null. Users who explicitly wrote the same defaults
// will see a one-time plan churn.
func approvalSettingsMatchesAPIDefaults(item approvalSettingsModel) bool {
	if item.Required.IsNull() || item.Required.ValueBool() {
		return false
	}
	if item.CanReviewOwnRequest.IsNull() || item.CanReviewOwnRequest.ValueBool() {
		return false
	}
	if item.MinNumApprovals.IsNull() || item.MinNumApprovals.ValueInt64() != 1 {
		return false
	}
	if item.CanApplyDeclinedChanges.IsNull() || !item.CanApplyDeclinedChanges.ValueBool() {
		return false
	}
	if !item.RequiredApprovalTags.IsNull() && len(item.RequiredApprovalTags.Elements()) > 0 {
		return false
	}
	if item.ServiceKind.IsNull() || item.ServiceKind.ValueString() != "launchdarkly" {
		return false
	}
	if !item.ServiceConfig.IsNull() && len(item.ServiceConfig.Elements()) > 0 {
		return false
	}
	if item.AutoApplyApprovedChanges.IsNull() || item.AutoApplyApprovedChanges.ValueBool() {
		return false
	}
	return true
}

func (r *ProjectResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	priorSchema := schema.Schema{Attributes: projectSchemaAttributesV0()}
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &priorSchema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior ProjectResourceModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}
				// v0 (SDKv2) stored default_client_side_availability as a
				// block (single-element list). v3 models it as a single
				// object — project the prior list accordingly.
				priorDCSA, d := csaObjectFromV0List(ctx, prior.DefaultClientSideAvailability, projectCSAAttrTypes)
				resp.Diagnostics.Append(d...)
				if resp.Diagnostics.HasError() {
					return
				}
				data := ProjectResourceModel{
					ID:                                   prior.ID,
					Key:                                  prior.Key,
					Name:                                 prior.Name,
					DefaultClientSideAvailability:        priorDCSA,
					Tags:                                 prior.Tags,
					Environments:                         prior.Environments,
					RequireViewAssociationForNewFlags:    prior.RequireViewAssociationForNewFlags,
					RequireViewAssociationForNewSegments: prior.RequireViewAssociationForNewSegments,
				}
				// IIS->DCSA migration: when prior state set include_in_snippet
				// and left default_client_side_availability empty, materialize
				// DCSA so the resource still controls the project default.
				// using_mobile_key was always implicitly true in v2 (see the
				// pre-removal Update patch that hardcoded UsingMobileKey: true).
				// When both were populated, drop IIS in favor of DCSA.
				dcsaEmpty := data.DefaultClientSideAvailability.IsNull() || data.DefaultClientSideAvailability.IsUnknown()
				iisSet := !prior.IncludeInSnippet.IsNull() && !prior.IncludeInSnippet.IsUnknown()
				if dcsaEmpty && iisSet {
					obj, d := types.ObjectValue(projectCSAAttrTypes, map[string]attr.Value{
						USING_ENVIRONMENT_ID: types.BoolValue(prior.IncludeInSnippet.ValueBool()),
						USING_MOBILE_KEY:     types.BoolValue(true),
					})
					resp.Diagnostics.Append(d...)
					data.DefaultClientSideAvailability = obj
				} else if featureFlagCSAMatchesAPIShape(ctx, data.DefaultClientSideAvailability) {
					// default_client_side_availability that matches LD account
					// defaults → null. Reuse the feature_flag helper since the
					// inner shape is identical.
					data.DefaultClientSideAvailability = types.ObjectNull(projectCSAAttrTypes)
				}
				// each environments[].approval_settings whose 1-element matches
				// API defaults → null. Decode envs, modify each, re-encode.
				if !data.Environments.IsNull() && !data.Environments.IsUnknown() && len(data.Environments.Elements()) > 0 {
					var envs []environmentModel
					resp.Diagnostics.Append(data.Environments.ElementsAs(ctx, &envs, false)...)
					if resp.Diagnostics.HasError() {
						return
					}
					approvalListType := types.ListType{ElemType: types.ObjectType{AttrTypes: frameworkApprovalSettingsObjectAttrTypes}}
					mutated := false
					for i := range envs {
						as := envs[i].ApprovalSettings
						if as.IsNull() || as.IsUnknown() || len(as.Elements()) != 1 {
							continue
						}
						var items []approvalSettingsModel
						d := as.ElementsAs(ctx, &items, false)
						if d.HasError() || len(items) != 1 {
							continue
						}
						if approvalSettingsMatchesAPIDefaults(items[0]) {
							envs[i].ApprovalSettings = types.ListNull(approvalListType.ElemType)
							mutated = true
						}
					}
					if mutated {
						envObjectType := types.ObjectType{AttrTypes: environmentAttrTypes}
						newList, d := types.ListValueFrom(ctx, envObjectType, envs)
						resp.Diagnostics.Append(d...)
						if !resp.Diagnostics.HasError() {
							data.Environments = newList
						}
					}
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			},
		},
	}
}

func (r *ProjectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

// ModifyPlan addresses environments-level sensitive Unknowns: nested-
// attribute schemas synthesize zero values for inner Computed fields
// once the user supplies the required ones (key/name/color). For
// api_key, mobile_key, client_side_id — secrets only LD can mint —
// this produces a "" plan value that Apply replaces with the real
// secret, tripping the framework's plan-vs-apply consistency check
// (see [[feedback-nested-attr-computed-sensitive]]). Mark these
// fields Unknown whenever there's no prior state entry for the same
// env key.
func (r *ProjectResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	if r.client == nil {
		return
	}
	var plan ProjectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ProjectResourceModel
	stateAbsent := req.State.Raw.IsNull()
	if !stateAbsent {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	envs, diags := markEnvSecretsUnknown(ctx, plan.Environments, state.Environments)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Environments = envs

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

// markEnvSecretsUnknown rewrites plan.Environments so api_key /
// mobile_key / client_side_id reflect the right env for each list
// position. The framework's UseStateForUnknown plan modifier on those
// inner attributes is index-based: when the user reorders envs, the
// modifier paints state[i]'s sensitive values onto plan[i] regardless
// of whether plan[i].key actually matches state[i].key. The result is
// post-Apply state pulling fresh API values that don't match the
// index-aligned plan and tripping the framework's plan-vs-apply
// consistency check.
//
// Fix: match by env key. For each plan env, if state has an env with
// the same key, use that state env's sensitive values; otherwise
// mark them Unknown so Apply can fill them in.
func markEnvSecretsUnknown(ctx context.Context, planList, stateList types.List) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	if planList.IsNull() || planList.IsUnknown() {
		return planList, diags
	}
	objType := types.ObjectType{AttrTypes: environmentAttrTypes}
	planEls := planList.Elements()
	if len(planEls) == 0 {
		return planList, diags
	}

	type envSecrets struct{ api, mobile, csid attr.Value }
	stateByKey := make(map[string]envSecrets)
	if !stateList.IsNull() && !stateList.IsUnknown() {
		for _, el := range stateList.Elements() {
			obj, ok := el.(basetypes.ObjectValue)
			if !ok {
				continue
			}
			a := obj.Attributes()
			keyVal, _ := a[KEY].(basetypes.StringValue)
			if keyVal.IsNull() || keyVal.IsUnknown() {
				continue
			}
			stateByKey[keyVal.ValueString()] = envSecrets{
				api:    a[API_KEY],
				mobile: a[MOBILE_KEY],
				csid:   a[CLIENT_SIDE_ID],
			}
		}
	}

	out := make([]attr.Value, 0, len(planEls))
	for _, el := range planEls {
		obj, ok := el.(basetypes.ObjectValue)
		if !ok {
			out = append(out, el)
			continue
		}
		attrs := obj.Attributes()
		keyVal, _ := attrs[KEY].(basetypes.StringValue)
		envKey := ""
		if !keyVal.IsNull() && !keyVal.IsUnknown() {
			envKey = keyVal.ValueString()
		}
		if secrets, ok := stateByKey[envKey]; ok {
			attrs[API_KEY] = secrets.api
			attrs[MOBILE_KEY] = secrets.mobile
			attrs[CLIENT_SIDE_ID] = secrets.csid
		} else {
			attrs[API_KEY] = types.StringUnknown()
			attrs[MOBILE_KEY] = types.StringUnknown()
			attrs[CLIENT_SIDE_ID] = types.StringUnknown()
		}
		newObj, d := types.ObjectValue(environmentAttrTypes, attrs)
		diags.Append(d...)
		out = append(out, newObj)
	}
	newList, d := types.ListValue(objType, out)
	diags.Append(d...)
	return newList, diags
}

// projectCSAValueFromAPI emits the CSA attribute matching the prior
// state shape: if prior was null, keep null so terraform doesn't see a
// populated state for a null plan; if prior was populated, emit the
// API's current values.
func projectCSAValueFromAPI(_ context.Context, csa *ldapi.ClientSideAvailability, prior basetypes.ObjectValue) (basetypes.ObjectValue, diag.Diagnostics) {
	priorEmpty := prior.IsNull() || prior.IsUnknown()
	if priorEmpty || csa == nil {
		return types.ObjectNull(projectCSAAttrTypes), nil
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
	return obj, diags
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
// view-association calls that follow PostProject. Used by both Create
// (state empty) and Update paths.
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

	csaPlanned := !plan.DefaultClientSideAvailability.IsNull() && !plan.DefaultClientSideAvailability.IsUnknown()
	csaChanged := isCreate || !plan.DefaultClientSideAvailability.Equal(state.DefaultClientSideAvailability)

	if csaPlanned && csaChanged {
		csa, d := csaPostFromObject(ctx, plan.DefaultClientSideAvailability)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
		patches = append(patches, patchReplace("/defaultClientSideAvailability", csa))
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
	stateEnvs := map[string]environmentModel{}
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

	tagsSet, d := setFromStringSlicePreservingPlan(ctx, project.Tags, data.Tags)
	diags.Append(d...)
	data.Tags = tagsSet

	csaObj, d := projectCSAValueFromAPI(ctx, project.DefaultClientSideAvailability, data.DefaultClientSideAvailability)
	diags.Append(d...)
	data.DefaultClientSideAvailability = csaObj

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

// csaPostFromObject extracts UsingEnvironmentId / UsingMobileKey from a
// framework client-side-availability object (shared by feature_flag's
// client_side_availability and project's default_client_side_availability).
func csaPostFromObject(ctx context.Context, o types.Object) (*ldapi.ClientSideAvailabilityPost, diag.Diagnostics) {
	if o.IsNull() || o.IsUnknown() {
		return nil, nil
	}
	type csaModel struct {
		UsingEnvironmentId types.Bool `tfsdk:"using_environment_id"`
		UsingMobileKey     types.Bool `tfsdk:"using_mobile_key"`
	}
	var m csaModel
	diags := o.As(ctx, &m, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, diags
	}
	return &ldapi.ClientSideAvailabilityPost{
		UsingEnvironmentId: m.UsingEnvironmentId.ValueBool(),
		UsingMobileKey:     m.UsingMobileKey.ValueBool(),
	}, diags
}
