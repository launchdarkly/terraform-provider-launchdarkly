package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v23"
)

var (
	_ resource.Resource                     = &SegmentResource{}
	_ resource.ResourceWithImportState      = &SegmentResource{}
	_ resource.ResourceWithModifyPlan       = &SegmentResource{}
	_ resource.ResourceWithConfigValidators = &SegmentResource{}
	_ resource.ResourceWithUpgradeState     = &SegmentResource{}
)

type SegmentResource struct {
	client *Client
}

type SegmentResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	ProjectKey           types.String `tfsdk:"project_key"`
	EnvKey               types.String `tfsdk:"env_key"`
	Key                  types.String `tfsdk:"key"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	Tags                 types.Set    `tfsdk:"tags"`
	CreationDate         types.Int64  `tfsdk:"creation_date"`
	Included             types.List   `tfsdk:"included"`
	Excluded             types.List   `tfsdk:"excluded"`
	IncludedContexts     types.List   `tfsdk:"included_contexts"`
	ExcludedContexts     types.List   `tfsdk:"excluded_contexts"`
	Rules                types.List   `tfsdk:"rules"`
	Unbounded            types.Bool   `tfsdk:"unbounded"`
	UnboundedContextKind types.String `tfsdk:"unbounded_context_kind"`
	ViewKeys             types.Set    `tfsdk:"view_keys"`
}

func NewSegmentResource() resource.Resource {
	return &SegmentResource{}
}

func (r *SegmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_segment"
}

func (r *SegmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: 1,
		Description: `Provides a LaunchDarkly segment resource.

This resource allows you to create and manage segments within your LaunchDarkly organization.

-> **Note:** When [segment approvals](https://launchdarkly.com/docs/home/releases/approvals) are enabled for an environment, segment **targeting** changes (` + "`included`" + `, ` + "`excluded`" + `, ` + "`rules`" + `, ` + "`included_contexts`" + `, ` + "`excluded_contexts`" + `) require approval. To let Terraform apply these changes non-interactively, grant its access token a custom role (` + "`launchdarkly_custom_role`" + `) that includes the ` + "`bypassRequiredSegmentApproval`" + ` action. Without that permission, ` + "`terraform apply`" + ` fails with an "approval is required" error. In that case, manage targeting through the approval workflow, for example with ` + "`lifecycle { ignore_changes = [included, excluded, rules] }`" + `, disable segment approvals for the environment, or switch to a token that has the bypass permission. A segment with no targeting can be created and managed regardless.`,
		Attributes: segmentSchemaAttributes(),
	}
}

func segmentSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		PROJECT_KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The segment's project key.", true),
			Validators:    []validator.String{keyValidator()},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		ENV_KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The segment's environment key.", true),
			Validators:    []validator.String{keyValidator()},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The unique key that references the segment.", true),
			Validators:    []validator.String{keyValidator()},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		NAME: schema.StringAttribute{
			Required:    true,
			Description: "The human-friendly name for the segment.",
		},
		DESCRIPTION: schema.StringAttribute{
			Optional:    true,
			Description: "The description of the segment's purpose.",
		},
		TAGS: schema.SetAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Validators:  []validator.Set{setvalidator.ValueStringsAre(tagValidator())},
			Description: "Tags associated with your resource.",
		},
		CREATION_DATE: schema.Int64Attribute{
			Computed:      true,
			Description:   "The segment's creation date represented as a UNIX epoch timestamp.",
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
		},
		INCLUDED: schema.ListAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: "List of user keys included in the segment. To target on other context kinds, use the included_contexts block attribute. This attribute is not valid when `unbounded` is set to `true`.",
		},
		EXCLUDED: schema.ListAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: "List of user keys excluded from the segment. To target on other context kinds, use the excluded_contexts block attribute. This attribute is not valid when `unbounded` is set to `true`.",
		},
		UNBOUNDED: schema.BoolAttribute{
			Optional:      true,
			Computed:      true,
			Default:       booldefault.StaticBool(false),
			Description:   addForceNewDescription("Whether to create a standard segment (`false`) or a big segment (`true`). Standard segments include rule-based and smaller list-based segments. Big segments include larger list-based segments and synced segments. Only use a big segment if you need to add more than 15,000 individual targets. It is not possible to manage the list of targeted contexts for big segments with Terraform.", true),
			PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
		},
		UNBOUNDED_CONTEXT_KIND: schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: addForceNewDescription("For big segments, the targeted context kind. If this attribute is not specified it defaults to `user`.", true),
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
				stringplanmodifier.RequiresReplace(),
			},
		},
		VIEW_KEYS: schema.SetAttribute{
			Optional:      true,
			Computed:      true,
			ElementType:   types.StringType,
			Description:   "A set of view keys to link this segment to. This is an alternative to using the `launchdarkly_view_links` resource for managing view associations. When set, this segment is linked to the specified views. The field is also computed, so Terraform reads back the current view associations from LaunchDarkly to detect drift. To explicitly remove all view associations, set `view_keys = []`. Removing the field from your configuration leaves existing associations unchanged. **Important**: Avoid using both `view_keys` and `launchdarkly_view_links` to manage the same segment. Mixed ownership can cause conflicts. When Terraform detects them, it logs a warning and reconciles to the configured `view_keys`. Choose one approach per resource.",
			Validators:    []validator.Set{},
			PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
		},
		INCLUDED_CONTEXTS: schema.ListNestedAttribute{
			Optional:    true,
			Description: "List of non-user target objects included in the segment. This attribute is not valid when `unbounded` is set to `true`.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					VALUES: schema.ListAttribute{
						Required:    true,
						ElementType: types.StringType,
						Description: "List of target object keys included in or excluded from the segment.",
					},
					CONTEXT_KIND: schema.StringAttribute{
						Required:    true,
						Description: "The context kind associated with this segment target. To target on user contexts, use the included and excluded attributes.",
					},
				},
			},
		},
		EXCLUDED_CONTEXTS: schema.ListNestedAttribute{
			Optional:    true,
			Description: "List of non-user target objects excluded from the segment. This attribute is not valid when `unbounded` is set to `true`.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					VALUES: schema.ListAttribute{
						Required:    true,
						ElementType: types.StringType,
						Description: "List of target object keys included in or excluded from the segment.",
					},
					CONTEXT_KIND: schema.StringAttribute{
						Required:    true,
						Description: "The context kind associated with this segment target. To target on user contexts, use the included and excluded attributes.",
					},
				},
			},
		},
		RULES: segmentRulesResourceAttribute(),
	}
}

func (r *SegmentResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	priorSchema := schema.Schema{Attributes: segmentSchemaAttributes()}
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &priorSchema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var data SegmentResourceModel
				resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
				if resp.Diagnostics.HasError() {
					return
				}
				data.Description = nullIfEmptyString(data.Description)
				data.UnboundedContextKind = nullIfEmptyString(data.UnboundedContextKind)
				data.Included = nullIfEmptyList(ctx, data.Included)
				data.Excluded = nullIfEmptyList(ctx, data.Excluded)
				data.IncludedContexts = nullIfEmptyList(ctx, data.IncludedContexts)
				data.ExcludedContexts = nullIfEmptyList(ctx, data.ExcludedContexts)
				data.Rules = nullIfEmptyList(ctx, data.Rules)
				data.Tags = nullIfEmptySet(ctx, data.Tags)
				data.ViewKeys = nullIfEmptySet(ctx, data.ViewKeys)
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			},
		},
	}
}

func (r *SegmentResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		segmentUnboundedConflictValidator{},
	}
}

// segmentUnboundedConflictValidator rejects configs where
// UNBOUNDED_CONTEXT_KIND is set alongside INCLUDED / EXCLUDED /
// INCLUDED_CONTEXTS / EXCLUDED_CONTEXTS / RULES.
type segmentUnboundedConflictValidator struct{}

func (v segmentUnboundedConflictValidator) Description(_ context.Context) string {
	return "unbounded_context_kind conflicts with included / excluded / included_contexts / excluded_contexts / rules"
}
func (v segmentUnboundedConflictValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (v segmentUnboundedConflictValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data SegmentResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if data.UnboundedContextKind.IsNull() || data.UnboundedContextKind.IsUnknown() || data.UnboundedContextKind.ValueString() == "" {
		return
	}
	forbiddenList := func(l types.List) bool {
		return !l.IsNull() && !l.IsUnknown() && len(l.Elements()) > 0
	}
	if forbiddenList(data.Included) || forbiddenList(data.Excluded) ||
		forbiddenList(data.IncludedContexts) || forbiddenList(data.ExcludedContexts) ||
		forbiddenList(data.Rules) {
		resp.Diagnostics.AddAttributeError(
			path.Root(UNBOUNDED_CONTEXT_KIND),
			"conflicting attributes",
			"unbounded_context_kind cannot be set together with included, excluded, included_contexts, excluded_contexts, or rules",
		)
	}
}

func (r *SegmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

// ModifyPlan ports customizeSegmentDiff: create-time view_keys validation
// when the project requires view association for new segments.
func (r *SegmentResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	if r.client == nil {
		return
	}
	if !req.State.Raw.IsNull() {
		return
	}
	var plan SegmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectKey := plan.ProjectKey.ValueString()
	if projectKey == "" {
		return
	}
	settings, err := getProjectViewSettings(ctx, r.client, projectKey)
	if err != nil {
		resp.Diagnostics.AddWarning(
			fmt.Sprintf("could not fetch project view settings for %q during plan", projectKey),
			err.Error(),
		)
		return
	}
	if !settings.RequireViewAssociationForNewSegments {
		return
	}
	if plan.ViewKeys.IsNull() || plan.ViewKeys.IsUnknown() || len(plan.ViewKeys.Elements()) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root(VIEW_KEYS),
			fmt.Sprintf("project %q requires new segments to be associated with at least one view. Please set the 'view_keys' attribute", projectKey),
			"",
		)
	}
}

func (r *SegmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SegmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectKey := plan.ProjectKey.ValueString()
	envKey := plan.EnvKey.ValueString()
	key := plan.Key.ValueString()

	if exists, err := projectExists(projectKey, r.client); !exists {
		if err != nil {
			resp.Diagnostics.AddError(err.Error(), "")
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("cannot find project with key %q", projectKey), "")
		return
	}
	if exists, err := environmentExists(projectKey, envKey, r.client); !exists {
		if err != nil {
			resp.Diagnostics.AddError(err.Error(), "")
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("environment %q not found in project %q — env_key must match the LaunchDarkly environment **key**. Create nested `environments` or a `launchdarkly_environment` first.", envKey, projectKey),
			"",
		)
		return
	}

	desc := plan.Description.ValueString()
	tags, d := stringSliceFromSet(ctx, plan.Tags)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	unbounded := plan.Unbounded.ValueBool()
	ubContextKind := plan.UnboundedContextKind.ValueString()
	viewKeysList, d := stringSliceFromSet(ctx, plan.ViewKeys)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	var err error
	if len(viewKeysList) > 0 {
		body := SegmentBodyWithViewKeys{
			Name:                 plan.Name.ValueString(),
			Key:                  key,
			Description:          desc,
			Tags:                 tags,
			Unbounded:            unbounded,
			UnboundedContextKind: ubContextKind,
			ViewKeys:             viewKeysList,
		}
		err = r.client.withConcurrency(ctx, func() error {
			return createSegmentWithViewKeys(ctx, r.client, projectKey, envKey, body)
		})
	} else {
		body := ldapi.SegmentBody{
			Name:                 plan.Name.ValueString(),
			Key:                  key,
			Description:          &desc,
			Tags:                 tags,
			Unbounded:            &unbounded,
			UnboundedContextKind: &ubContextKind,
		}
		err = r.client.withConcurrency(ctx, func() error {
			_, _, e := r.client.ld.SegmentsApi.PostSegment(r.client.ctx, projectKey, envKey).SegmentBody(body).Execute()
			return e
		})
	}
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to create segment %q in project %q: %s", key, projectKey, handleLdapiErr(err).Error()), "")
		return
	}

	if d := r.applySegmentUpdate(ctx, plan, SegmentResourceModel{}, true); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}
	r.readIntoModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = types.StringValue(projectKey + "/" + envKey + "/" + key)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SegmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SegmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readIntoModel(ctx, &data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SegmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state SegmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d := r.applySegmentUpdate(ctx, plan, state, false); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}
	r.readIntoModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = types.StringValue(plan.ProjectKey.ValueString() + "/" + plan.EnvKey.ValueString() + "/" + plan.Key.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SegmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SegmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, e := r.client.ld.SegmentsApi.DeleteSegment(r.client.ctx, data.ProjectKey.ValueString(), data.EnvKey.ValueString(), data.Key.ValueString()).Execute()
		return e
	})
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to delete segment %q from project %q: %s", data.Key.ValueString(), data.ProjectKey.ValueString(), handleLdapiErr(err).Error()), "")
	}
}

func (r *SegmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if strings.Count(req.ID, "/") != 2 {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("expected project_key/env_key/segment_key, got %q", req.ID))
		return
	}
	parts := strings.SplitN(req.ID, "/", 3)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(ENV_KEY), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(KEY), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *SegmentResource) applySegmentUpdate(ctx context.Context, plan, state SegmentResourceModel, isCreate bool) diag.Diagnostics {
	var diags diag.Diagnostics
	projectKey := plan.ProjectKey.ValueString()
	envKey := plan.EnvKey.ValueString()
	key := plan.Key.ValueString()

	desc := plan.Description.ValueString()
	tags, d := stringSliceFromSet(ctx, plan.Tags)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	included, d := stringSliceFromList(ctx, plan.Included)
	diags.Append(d...)
	excluded, d := stringSliceFromList(ctx, plan.Excluded)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	includedContexts, d := segmentTargetsFromList(ctx, plan.IncludedContexts)
	diags.Append(d...)
	excludedContexts, d := segmentTargetsFromList(ctx, plan.ExcludedContexts)
	diags.Append(d...)
	rules, d := segmentRulesFromList(ctx, plan.Rules)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	comment := "Terraform"
	var patchOps []ldapi.PatchOperation
	if isCreate {
		// The shell created by PostSegment / createSegmentWithViewKeys already
		// carries name, description, tags and unbounded settings, so a create
		// only needs to PATCH targeting. We deliberately omit the no-op
		// targeting replaces (and the legacy "/temporary" op, which targets a
		// field segments don't have): when segment approvals are enabled those
		// ops are themselves gated, so sending them would make even a
		// targeting-free segment fail to create (issue #370). A segment with no
		// targeting therefore needs no PATCH at all and is created by the shell
		// POST alone.
		patchOps = appendSegmentTargetingOps(nil, included, excluded, rules, includedContexts, excludedContexts)
	} else {
		patchOps = []ldapi.PatchOperation{
			patchReplace("/name", plan.Name.ValueString()),
			patchReplace("/description", desc),
			patchReplace("/included", included),
			patchReplace("/excluded", excluded),
			patchReplace("/rules", rules),
			patchReplace("/includedContexts", includedContexts),
			patchReplace("/excludedContexts", excludedContexts),
		}
		if !plan.Tags.Equal(state.Tags) && len(tags) == 0 {
			patchOps = append(patchOps, patchRemove("/tags"))
		} else {
			patchOps = append(patchOps, patchReplace("/tags", tags))
		}
	}

	if len(patchOps) > 0 {
		err := r.client.withConcurrency(r.client.ctx, func() error {
			_, _, e := r.client.ld.SegmentsApi.PatchSegment(r.client.ctx, projectKey, envKey, key).PatchWithComment(ldapi.PatchWithComment{
				Comment: &comment,
				Patch:   patchOps,
			}).Execute()
			return e
		})
		if err != nil {
			if isApprovalRequiredErr(err) {
				diags.Append(r.segmentApprovalRequiredDiag(isCreate, projectKey, envKey, key))
				return diags
			}
			diags.AddError(fmt.Sprintf("failed to update segment %q in project %q: %s", key, projectKey, handleLdapiErr(err).Error()), "")
			return diags
		}
	}

	// View association reconciliation.
	viewKeysChanged := isCreate || !plan.ViewKeys.Equal(state.ViewKeys)
	if !viewKeysChanged {
		return diags
	}
	desiredViews, d := stringSliceFromSet(ctx, plan.ViewKeys)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	betaClient, err := newBetaClient(r.client.apiKey, r.client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		diags.AddError(fmt.Sprintf("failed to create beta client for view linking: %v", err), "")
		return diags
	}
	var env *ldapi.Environment
	err = r.client.withConcurrency(r.client.ctx, func() error {
		env, _, err = r.client.ld.EnvironmentsApi.GetEnvironment(r.client.ctx, projectKey, envKey).Execute()
		return err
	})
	if err != nil {
		diags.AddError(fmt.Sprintf("failed to get environment %q in project %q: %s", envKey, projectKey, handleLdapiErr(err).Error()), "")
		return diags
	}

	if plan.ViewKeys.IsNull() || plan.ViewKeys.IsUnknown() {
		// Unlink any views previously managed by this resource.
		oldKeys, _ := stringSliceFromSet(ctx, state.ViewKeys)
		for _, vk := range oldKeys {
			if err := unlinkSegmentsFromView(betaClient, projectKey, vk, []ViewSegmentIdentifier{{EnvironmentId: env.Id, SegmentKey: key}}); err != nil {
				diags.AddError(fmt.Sprintf("failed to unlink segment %q from view %q: %v", key, vk, err), "")
				return diags
			}
		}
		return diags
	}

	for _, vk := range desiredViews {
		exists, vErr := viewExists(projectKey, vk, betaClient)
		if vErr != nil {
			diags.AddError(fmt.Sprintf("failed to check if view %q exists: %v", vk, vErr), "")
			return diags
		}
		if !exists {
			diags.AddError(fmt.Sprintf("cannot link segment to view %q in project %q: view does not exist", vk, projectKey), "")
			return diags
		}
	}
	currentViews, vErr := getViewsContainingSegment(betaClient, projectKey, env.Id, key)
	if vErr != nil {
		log.Printf("[WARN] failed to get current views for segment %q: %v", key, vErr)
		currentViews = []string{}
	}
	toAdd := difference(desiredViews, currentViews)
	toRemove := difference(currentViews, desiredViews)
	for _, vk := range toRemove {
		if err := unlinkSegmentsFromView(betaClient, projectKey, vk, []ViewSegmentIdentifier{{EnvironmentId: env.Id, SegmentKey: key}}); err != nil {
			diags.AddError(fmt.Sprintf("failed to unlink segment %q from view %q: %v", key, vk, err), "")
			return diags
		}
	}
	for _, vk := range toAdd {
		if err := linkSegmentsToView(betaClient, projectKey, vk, []ViewSegmentIdentifier{{EnvironmentId: env.Id, SegmentKey: key}}); err != nil {
			diags.AddError(fmt.Sprintf("failed to link segment %q to view %q: %v", key, vk, err), "")
			return diags
		}
	}
	return diags
}

// appendSegmentTargetingOps appends a replace op for each segment targeting
// collection that is non-empty. Empty collections are skipped because a
// freshly created segment shell already has them empty, and replacing them
// with an empty value is gated when segment approvals are enabled (issue
// #370) — sending such no-op replaces would needlessly fail an otherwise
// targeting-free create.
func appendSegmentTargetingOps(ops []ldapi.PatchOperation, included, excluded []string, rules []ldapi.UserSegmentRule, includedContexts, excludedContexts []ldapi.SegmentTarget) []ldapi.PatchOperation {
	if len(included) > 0 {
		ops = append(ops, patchReplace("/included", included))
	}
	if len(excluded) > 0 {
		ops = append(ops, patchReplace("/excluded", excluded))
	}
	if len(rules) > 0 {
		ops = append(ops, patchReplace("/rules", rules))
	}
	if len(includedContexts) > 0 {
		ops = append(ops, patchReplace("/includedContexts", includedContexts))
	}
	if len(excludedContexts) > 0 {
		ops = append(ops, patchReplace("/excludedContexts", excludedContexts))
	}
	return ops
}

// segmentApprovalRequiredDiag builds the diagnostic returned when a segment
// PATCH is rejected because segment approvals are enabled for the environment
// and the token's role does not permit bypassing them. Terraform applies
// changes non-interactively, so it cannot satisfy an inline approval; the fix
// is to grant the token a custom role with the "bypassRequiredSegmentApproval"
// action. On create the function also rolls back the shell that PostSegment
// created (DELETE is not gated by approvals) so a retry does not collide with
// an orphaned segment. See issue #370.
func (r *SegmentResource) segmentApprovalRequiredDiag(isCreate bool, projectKey, envKey, key string) diag.Diagnostic {
	verb := "updated"
	remediation := "Grant this token a custom role that includes the \"bypassRequiredSegmentApproval\" action so Terraform can apply targeting changes directly, remove the targeting attributes (included / excluded / rules / included_contexts / excluded_contexts) from this resource, or disable segment approvals for this environment."
	rollback := ""
	if isCreate {
		verb = "created"
		remediation = "Grant this token a custom role that includes the \"bypassRequiredSegmentApproval\" action so Terraform can create the segment with its targeting directly, remove the targeting attributes (included / excluded / rules / included_contexts / excluded_contexts) so Terraform creates only the segment shell and you manage targeting through the approval workflow, or disable segment approvals for this environment."
		delErr := r.client.withConcurrency(r.client.ctx, func() error {
			_, e := r.client.ld.SegmentsApi.DeleteSegment(r.client.ctx, projectKey, envKey, key).Execute()
			return e
		})
		if delErr == nil {
			rollback = " The partially created segment was rolled back to keep Terraform state consistent."
		} else {
			rollback = fmt.Sprintf(" Note: the partially created segment %q could not be removed automatically (%s); delete it manually before retrying.", key, handleLdapiErr(delErr).Error())
		}
	}
	return diag.NewErrorDiagnostic(
		fmt.Sprintf("segment %q cannot be %s in project %q: segment approvals are enabled for environment %q", key, verb, projectKey, envKey),
		fmt.Sprintf("LaunchDarkly rejected the segment targeting change with \"approval is required\" (HTTP 403). Segment approvals gate targeting changes in this environment, and this token's role does not permit bypassing them. Because Terraform applies changes non-interactively, it cannot satisfy an inline approval; grant the token a custom role with the \"bypassRequiredSegmentApproval\" action to let it apply these changes directly.%s\n\n%s", rollback, remediation),
	)
}

func (r *SegmentResource) readIntoModel(ctx context.Context, data *SegmentResourceModel, diags *diag.Diagnostics) {
	projectKey := data.ProjectKey.ValueString()
	envKey := data.EnvKey.ValueString()
	key := data.Key.ValueString()

	var segment *ldapi.UserSegment
	var res *http.Response
	var err error
	err = r.client.withConcurrency(r.client.ctx, func() error {
		segment, res, err = r.client.ld.SegmentsApi.GetSegment(r.client.ctx, projectKey, envKey, key).Execute()
		return err
	})
	if isStatusNotFound(res) {
		data.ID = types.StringNull()
		return
	}
	if err != nil {
		diags.AddError(fmt.Sprintf("failed to get segment %q of project %q: %s", key, projectKey, handleLdapiErr(err).Error()), "")
		return
	}

	data.ID = types.StringValue(projectKey + "/" + envKey + "/" + key)
	data.Name = types.StringValue(segment.Name)
	if segment.Description != nil {
		data.Description = stringValueOrNull(*segment.Description)
	} else {
		data.Description = types.StringNull()
	}
	data.CreationDate = types.Int64Value(segment.CreationDate)

	tagsSet, d := setFromStringSlicePreservingPlan(ctx, segment.Tags, data.Tags)
	diags.Append(d...)
	data.Tags = tagsSet

	if segment.Unbounded != nil {
		data.Unbounded = types.BoolValue(*segment.Unbounded)
	} else {
		data.Unbounded = types.BoolValue(false)
	}
	if segment.UnboundedContextKind != nil && *segment.UnboundedContextKind != "" {
		data.UnboundedContextKind = types.StringValue(*segment.UnboundedContextKind)
	} else {
		data.UnboundedContextKind = types.StringValue("")
	}

	includedList, d := listFromStringSlicePreservingPlan(ctx, segment.Included, data.Included)
	diags.Append(d...)
	data.Included = includedList
	excludedList, d := listFromStringSlicePreservingPlan(ctx, segment.Excluded, data.Excluded)
	diags.Append(d...)
	data.Excluded = excludedList

	data.IncludedContexts = segmentTargetsToFrameworkListImpl(ctx, segment.IncludedContexts)
	data.ExcludedContexts = segmentTargetsToFrameworkListImpl(ctx, segment.ExcludedContexts)
	data.Rules = segmentResourceRulesValue(ctx, segment.Rules, diags)

	// View association reads — best-effort.
	betaClient, bcErr := newBetaClient(r.client.apiKey, r.client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if bcErr != nil {
		log.Printf("[WARN] failed to create beta client for view lookup: %v", bcErr)
		if data.ViewKeys.IsNull() || data.ViewKeys.IsUnknown() {
			data.ViewKeys = types.SetValueMust(types.StringType, []attr.Value{})
		}
		return
	}
	var env *ldapi.Environment
	err = r.client.withConcurrency(r.client.ctx, func() error {
		env, _, err = r.client.ld.EnvironmentsApi.GetEnvironment(r.client.ctx, projectKey, envKey).Execute()
		return err
	})
	if err != nil {
		log.Printf("[WARN] failed to get environment %q in project %q: %v", envKey, projectKey, err)
		data.ViewKeys = types.SetValueMust(types.StringType, []attr.Value{})
		return
	}
	viewKeys, vErr := getViewsContainingSegment(betaClient, projectKey, env.Id, key)
	if vErr != nil {
		log.Printf("[WARN] failed to get views for segment %q: %v", key, vErr)
		viewKeys = []string{}
	}
	viewKeysSet, d := setFromStringSlice(ctx, viewKeys)
	diags.Append(d...)
	data.ViewKeys = viewKeysSet
}

// segmentResourceRulesValue is the resource-side analogue of
// segmentRulesToFrameworkList (which the segment data source uses).
// The data source declares weight / bucket_by / rollout_context_kind
// as Computed-only and tolerates zero values; the resource declares
// them Optional-only and must emit null when the API returned nil/zero
// to satisfy terraform-core's plan-apply consistency check.
// rollout_context_kind is Optional+Computed+Default("user") at the
// schema level so plan and state both end up at "user" when the user
// omits it.
func segmentResourceRulesValue(ctx context.Context, rules []ldapi.UserSegmentRule, diags *diag.Diagnostics) types.List {
	objectType := types.ObjectType{AttrTypes: segmentRuleAttrTypes}
	elements := make([]attr.Value, 0, len(rules))
	for _, r := range rules {
		clauses, d := frameworkClausesValue(ctx, r.Clauses)
		diags.Append(d...)
		weight := types.Int64Null()
		if r.Weight != nil && *r.Weight > 0 {
			weight = types.Int64Value(int64(*r.Weight))
		}
		rckValue := stringValueOrNullFromPointer(r.RolloutContextKind)
		if rckValue.IsNull() {
			rckValue = types.StringValue("user")
		}
		obj, d := types.ObjectValue(segmentRuleAttrTypes, map[string]attr.Value{
			CLAUSES:              clauses,
			WEIGHT:               weight,
			BUCKET_BY:            stringValueOrNullFromPointer(r.BucketBy),
			ROLLOUT_CONTEXT_KIND: rckValue,
		})
		diags.Append(d...)
		elements = append(elements, obj)
	}
	if len(elements) == 0 {
		return types.ListNull(objectType)
	}
	list, d := types.ListValue(objectType, elements)
	diags.Append(d...)
	return list
}

// segmentRulesResourceAttribute declares the framework attribute schema
// for segment rules.
func segmentRulesResourceAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Optional:    true,
		Description: "List of custom rules to apply to the segment. This attribute is not valid when `unbounded` is set to `true`.",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				WEIGHT: schema.Int64Attribute{
					Optional:    true,
					Description: "The integer weight of the rule (between 1 and 100000).",
				},
				BUCKET_BY: schema.StringAttribute{
					Optional:    true,
					Description: "The attribute by which to group contexts together.",
				},
				ROLLOUT_CONTEXT_KIND: schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Default:     stringdefault.StaticString("user"),
					Description: "The context kind associated with this segment rule. This argument is only valid if `weight` is also specified. If omitted, defaults to `user`.",
				},
				CLAUSES: frameworkClausesResourceAttribute(),
			},
		},
	}
}

// segmentTargetsFromList converts a framework ListValue of
// included/excluded contexts into ldapi.SegmentTarget slices.
func segmentTargetsFromList(ctx context.Context, list types.List) ([]ldapi.SegmentTarget, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return []ldapi.SegmentTarget{}, diags
	}
	type targetModel struct {
		Values      types.List   `tfsdk:"values"`
		ContextKind types.String `tfsdk:"context_kind"`
	}
	var models []targetModel
	d := list.ElementsAs(ctx, &models, false)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]ldapi.SegmentTarget, 0, len(models))
	for _, m := range models {
		vals, d := stringSliceFromList(ctx, m.Values)
		diags.Append(d...)
		ck := m.ContextKind.ValueString()
		out = append(out, ldapi.SegmentTarget{
			Values:      vals,
			ContextKind: &ck,
		})
	}
	return out, diags
}

// segmentRulesFromList converts a framework ListValue of segment rules
// into []ldapi.UserSegmentRule.
func segmentRulesFromList(ctx context.Context, list types.List) ([]ldapi.UserSegmentRule, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return []ldapi.UserSegmentRule{}, diags
	}
	type ruleModel struct {
		Clauses            types.List   `tfsdk:"clauses"`
		Weight             types.Int64  `tfsdk:"weight"`
		BucketBy           types.String `tfsdk:"bucket_by"`
		RolloutContextKind types.String `tfsdk:"rollout_context_kind"`
	}
	var models []ruleModel
	d := list.ElementsAs(ctx, &models, false)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]ldapi.UserSegmentRule, 0, len(models))
	for _, m := range models {
		clauses, d := frameworkClausesFromList(ctx, m.Clauses)
		diags.Append(d...)
		r := ldapi.NewUserSegmentRule(clauses)
		bucketBy := m.BucketBy.ValueString()
		if bucketBy != "" {
			r.SetBucketBy(bucketBy)
		}
		w := int32(m.Weight.ValueInt64())
		if w > 0 {
			r.SetWeight(w)
			rck := m.RolloutContextKind.ValueString()
			if rck != "" {
				r.SetRolloutContextKind(rck)
			}
		}
		out = append(out, *r)
	}
	return out, diags
}
