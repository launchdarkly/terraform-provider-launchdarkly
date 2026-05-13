package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"time"

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
)

// -----------------------------------------------------------------------------
// launchdarkly_view_links
// -----------------------------------------------------------------------------

var (
	_ resource.Resource                = &ViewLinksResource{}
	_ resource.ResourceWithImportState = &ViewLinksResource{}
)

type ViewLinksResource struct {
	client *Client
	beta   *Client
}

type ViewLinksResourceModel struct {
	ID         types.String `tfsdk:"id"`
	ProjectKey types.String `tfsdk:"project_key"`
	ViewKey    types.String `tfsdk:"view_key"`
	Flags      types.Set    `tfsdk:"flags"`
	Segments   types.Set    `tfsdk:"segments"`
}

func NewViewLinksResource() resource.Resource { return &ViewLinksResource{} }

func (r *ViewLinksResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_view_links"
}

var viewLinkSegmentAttrTypes = map[string]attr.Type{
	SEGMENT_ENVIRONMENT_ID: types.StringType,
	SEGMENT_KEY:            types.StringType,
}

func (r *ViewLinksResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: viewLinksDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:      true,
				Description:   "The project key. A change in this field will force the destruction of the existing resource and the creation of a new one.",
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			VIEW_KEY: schema.StringAttribute{
				Required:      true,
				Description:   "The view key to link resources to. A change in this field will force the destruction of the existing resource and the creation of a new one.",
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			FLAGS: schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "A set of feature flag keys to link to the view.",
			},
		},
		Blocks: map[string]schema.Block{
			SEGMENTS: schema.SetNestedBlock{
				Description: "A set of segments to link to the view. Each segment is identified by its environment ID and segment key.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						SEGMENT_ENVIRONMENT_ID: schema.StringAttribute{
							Required:    true,
							Description: "The environment ID of the segment.",
							Validators:  []validator.String{idValidator()},
						},
						SEGMENT_KEY: schema.StringAttribute{
							Required:    true,
							Description: "The key of the segment.",
							Validators:  []validator.String{keyValidator()},
						},
					},
				},
			},
		},
	}
}

func (r *ViewLinksResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
	if r.client == nil {
		return
	}
	beta, err := newBetaClient(r.client.apiKey, r.client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build LaunchDarkly beta client", err.Error())
		return
	}
	r.beta = beta
}

func (r *ViewLinksResource) betaClient() (*Client, error) {
	if r.beta != nil {
		return r.beta, nil
	}
	return newBetaClient(r.client.apiKey, r.client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
}

func (r *ViewLinksResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ViewLinksResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	betaClient, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	viewKey := plan.ViewKey.ValueString()

	exists, err := viewExists(projectKey, viewKey, betaClient)
	if err != nil {
		resp.Diagnostics.AddError("Failed to check view", err.Error())
		return
	}
	if !exists {
		resp.Diagnostics.AddError("View not found", fmt.Sprintf("cannot find view with key %q in project %q", viewKey, projectKey))
		return
	}

	flags, d := stringSliceFromSet(ctx, plan.Flags)
	resp.Diagnostics.Append(d...)
	segments, d := viewSegmentIdentifiersFromSet(ctx, plan.Segments)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(flags) > 0 {
		if err := linkResourcesToView(betaClient, projectKey, viewKey, FLAGS, flags); err != nil {
			resp.Diagnostics.AddError("Failed to link flags to view", err.Error())
			return
		}
	}
	if len(segments) > 0 {
		if err := linkSegmentsToView(betaClient, projectKey, viewKey, segments); err != nil {
			resp.Diagnostics.AddError("Failed to link segments to view", err.Error())
			return
		}
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/%s", projectKey, viewKey))
	r.readIntoModel(ctx, betaClient, projectKey, viewKey, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ViewLinksResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ViewLinksResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	betaClient, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}
	projectKey := data.ProjectKey.ValueString()
	viewKey := data.ViewKey.ValueString()

	exists, err := viewExists(projectKey, viewKey, betaClient)
	if err != nil {
		resp.Diagnostics.AddError("Failed to check view", err.Error())
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}
	r.readIntoModel(ctx, betaClient, projectKey, viewKey, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ViewLinksResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ViewLinksResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	betaClient, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}
	projectKey := plan.ProjectKey.ValueString()
	viewKey := plan.ViewKey.ValueString()

	if !plan.Flags.Equal(state.Flags) {
		oldFlags, d := stringSliceFromSet(ctx, state.Flags)
		resp.Diagnostics.Append(d...)
		newFlags, d := stringSliceFromSet(ctx, plan.Flags)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		toAdd := stringSliceDifference(newFlags, oldFlags)
		toRemove := stringSliceDifference(oldFlags, newFlags)
		if len(toRemove) > 0 {
			if err := unlinkResourcesFromView(betaClient, projectKey, viewKey, FLAGS, toRemove); err != nil {
				resp.Diagnostics.AddError("Failed to unlink flags", err.Error())
				return
			}
		}
		if len(toAdd) > 0 {
			if err := linkResourcesToView(betaClient, projectKey, viewKey, FLAGS, toAdd); err != nil {
				resp.Diagnostics.AddError("Failed to link flags", err.Error())
				return
			}
		}
	}

	if !plan.Segments.Equal(state.Segments) {
		oldSegs, d := viewSegmentIdentifiersFromSet(ctx, state.Segments)
		resp.Diagnostics.Append(d...)
		newSegs, d := viewSegmentIdentifiersFromSet(ctx, plan.Segments)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		toAdd := differenceSegmentIdentifiers(newSegs, oldSegs)
		toRemove := differenceSegmentIdentifiers(oldSegs, newSegs)
		if len(toRemove) > 0 {
			if err := unlinkSegmentsFromView(betaClient, projectKey, viewKey, toRemove); err != nil {
				resp.Diagnostics.AddError("Failed to unlink segments", err.Error())
				return
			}
		}
		if len(toAdd) > 0 {
			if err := linkSegmentsToView(betaClient, projectKey, viewKey, toAdd); err != nil {
				resp.Diagnostics.AddError("Failed to link segments", err.Error())
				return
			}
		}
	}

	r.readIntoModel(ctx, betaClient, projectKey, viewKey, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ViewLinksResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ViewLinksResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	betaClient, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}
	projectKey := data.ProjectKey.ValueString()
	viewKey := data.ViewKey.ValueString()

	flags, d := stringSliceFromSet(ctx, data.Flags)
	resp.Diagnostics.Append(d...)
	segments, d := viewSegmentIdentifiersFromSet(ctx, data.Segments)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(flags) > 0 {
		if err := unlinkResourcesFromView(betaClient, projectKey, viewKey, FLAGS, flags); err != nil {
			resp.Diagnostics.AddError("Failed to unlink flags", err.Error())
			return
		}
	}
	if len(segments) > 0 {
		if err := unlinkSegmentsFromView(betaClient, projectKey, viewKey, segments); err != nil {
			resp.Diagnostics.AddError("Failed to unlink segments", err.Error())
			return
		}
	}
}

func (r *ViewLinksResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectKey, viewKey, err := viewIdToKeys(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), projectKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(VIEW_KEY), viewKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *ViewLinksResource) readIntoModel(
	ctx context.Context,
	betaClient *Client,
	projectKey, viewKey string,
	data *ViewLinksResourceModel,
	diags *diag.Diagnostics,
) {
	linkedFlags, err := getLinkedResources(betaClient, projectKey, viewKey, FLAGS)
	if err != nil {
		diags.AddError("Failed to get linked flags", err.Error())
		return
	}
	flagKeys := make([]string, len(linkedFlags))
	for i, f := range linkedFlags {
		flagKeys[i] = f.ResourceKey
	}
	flagsSet, d := setFromStringSlice(ctx, flagKeys)
	diags.Append(d...)
	data.Flags = flagsSet

	linkedSegments, err := getLinkedResources(betaClient, projectKey, viewKey, SEGMENTS)
	if err != nil {
		diags.AddError("Failed to get linked segments", err.Error())
		return
	}
	segObjType := types.ObjectType{AttrTypes: viewLinkSegmentAttrTypes}
	segElems := make([]attr.Value, 0, len(linkedSegments))
	for _, s := range linkedSegments {
		obj, d := types.ObjectValue(viewLinkSegmentAttrTypes, map[string]attr.Value{
			SEGMENT_ENVIRONMENT_ID: types.StringValue(s.EnvironmentId),
			SEGMENT_KEY:            types.StringValue(s.ResourceKey),
		})
		diags.Append(d...)
		segElems = append(segElems, obj)
	}
	segSet, d := types.SetValue(segObjType, segElems)
	diags.Append(d...)
	data.Segments = segSet
	data.ID = types.StringValue(fmt.Sprintf("%s/%s", projectKey, viewKey))
}

// viewSegmentIdentifiersFromSet converts a framework set of nested
// objects into a []ViewSegmentIdentifier.
func viewSegmentIdentifiersFromSet(ctx context.Context, set types.Set) ([]ViewSegmentIdentifier, diag.Diagnostics) {
	var diags diag.Diagnostics
	if set.IsNull() || set.IsUnknown() {
		return nil, diags
	}
	type seg struct {
		EnvironmentID string `tfsdk:"environment_id"`
		SegmentKey    string `tfsdk:"segment_key"`
	}
	var raw []seg
	diags.Append(set.ElementsAs(ctx, &raw, false)...)
	out := make([]ViewSegmentIdentifier, len(raw))
	for i, s := range raw {
		out[i] = ViewSegmentIdentifier{EnvironmentId: s.EnvironmentID, SegmentKey: s.SegmentKey}
	}
	return out, diags
}

func stringSliceDifference(a, b []string) []string {
	seen := make(map[string]struct{}, len(b))
	for _, s := range b {
		seen[s] = struct{}{}
	}
	var out []string
	for _, s := range a {
		if _, ok := seen[s]; !ok {
			out = append(out, s)
		}
	}
	return out
}

const viewLinksDescription = `Provides a LaunchDarkly view links resource for managing bulk resource linkage to views.

-> **Note:** Views are available to customers on an Enterprise LaunchDarkly plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).

~> **Beta:** This resource uses a beta API. Beta resources may change or be removed in future versions.

This resource allows you to efficiently link multiple flags and/or segments to a specific view. This is particularly useful for administrators organizing resources by team or deployment unit.

-> **Note:** This resource manages ALL links for the specified resource types within a view. Adding or removing items from the configuration will link or unlink those resources accordingly.

-> **Warning:** Do not use both ` + "`view_links`" + ` and ` + "`view_keys`" + ` to manage the same flag or segment's view associations. Mixed ownership can cause conflicts; when detected, Terraform logs a warning and reconciles to the managing resource's configured associations. Choose one approach per resource.`

// -----------------------------------------------------------------------------
// launchdarkly_view_filter_links
// -----------------------------------------------------------------------------

var (
	_ resource.Resource                     = &ViewFilterLinksResource{}
	_ resource.ResourceWithImportState      = &ViewFilterLinksResource{}
	_ resource.ResourceWithConfigValidators = &ViewFilterLinksResource{}
	_ resource.ResourceWithModifyPlan       = &ViewFilterLinksResource{}
)

type ViewFilterLinksResource struct {
	client *Client
	beta   *Client
}

type ViewFilterLinksResourceModel struct {
	ID                         types.String `tfsdk:"id"`
	ProjectKey                 types.String `tfsdk:"project_key"`
	ViewKey                    types.String `tfsdk:"view_key"`
	FlagFilter                 types.String `tfsdk:"flag_filter"`
	SegmentFilter              types.String `tfsdk:"segment_filter"`
	SegmentFilterEnvironmentID types.String `tfsdk:"segment_filter_environment_id"`
	ReconcileOnApply           types.Bool   `tfsdk:"reconcile_on_apply"`
	ResolvedAt                 types.String `tfsdk:"resolved_at"`
}

func NewViewFilterLinksResource() resource.Resource { return &ViewFilterLinksResource{} }

func (r *ViewFilterLinksResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_view_filter_links"
}

func (r *ViewFilterLinksResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: viewFilterLinksDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:      true,
				Description:   "The project key. A change in this field will force the destruction of the existing resource and the creation of a new one.",
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			VIEW_KEY: schema.StringAttribute{
				Required:      true,
				Description:   "The view key to link resources to. A change in this field will force the destruction of the existing resource and the creation of a new one.",
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			FLAG_FILTER: schema.StringAttribute{
				Optional:    true,
				Description: "A filter expression to match feature flags for linking to the view. Uses the same filter syntax as the flag list API endpoint (e.g. `tags:frontend`, `status:active`).",
			},
			SEGMENT_FILTER: schema.StringAttribute{
				Optional:    true,
				Description: "A filter expression to match segments for linking to the view. Uses the segment query filter syntax (e.g. `tags anyOf [\"backend\"]`, `query = \"my-segment\"`, `unbounded = true`). Requires `segment_filter_environment_id` to be set.",
			},
			SEGMENT_FILTER_ENVIRONMENT_ID: schema.StringAttribute{
				Optional:    true,
				Description: "The environment ID to use when resolving segment filters. Required when `segment_filter` is set. This is the environment's opaque ID (e.g. from `launchdarkly_project.environments[*].client_side_id`).",
				Validators:  []validator.String{idValidator()},
			},
			RECONCILE_ON_APPLY: schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to re-resolve configured filters on every `terraform apply` even when no resource arguments changed. When true, Terraform will show an in-place update on each apply and `resolved_at` will change every run.",
			},
			RESOLVED_AT: schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of the last successful filter resolution. This value updates when the resource is created or updated, and on every apply when `reconcile_on_apply` is true.",
			},
		},
	}
}

func (r *ViewFilterLinksResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{viewFilterLinksValidator{}}
}

type viewFilterLinksValidator struct{}

func (viewFilterLinksValidator) Description(context.Context) string {
	return "flag_filter or segment_filter required; segment_filter requires segment_filter_environment_id"
}
func (viewFilterLinksValidator) MarkdownDescription(context.Context) string { return "" }
func (viewFilterLinksValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ViewFilterLinksResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	flagSet := !data.FlagFilter.IsNull() && !data.FlagFilter.IsUnknown() && data.FlagFilter.ValueString() != ""
	segSet := !data.SegmentFilter.IsNull() && !data.SegmentFilter.IsUnknown() && data.SegmentFilter.ValueString() != ""
	envSet := !data.SegmentFilterEnvironmentID.IsNull() && !data.SegmentFilterEnvironmentID.IsUnknown() && data.SegmentFilterEnvironmentID.ValueString() != ""
	if !flagSet && !segSet {
		resp.Diagnostics.AddError(
			"Missing filter",
			"at least one of flag_filter or segment_filter must be set",
		)
	}
	if segSet && !envSet {
		resp.Diagnostics.AddAttributeError(
			path.Root(SEGMENT_FILTER_ENVIRONMENT_ID),
			"segment_filter requires segment_filter_environment_id",
			"segment_filter is set; segment_filter_environment_id must also be set",
		)
	}
	if envSet && !segSet {
		resp.Diagnostics.AddAttributeError(
			path.Root(SEGMENT_FILTER),
			"segment_filter_environment_id requires segment_filter",
			"segment_filter_environment_id is set; segment_filter must also be set",
		)
	}
}

// ModifyPlan re-marks resolved_at unknown when reconcile_on_apply is
// true, so the framework computes a new value each apply.
func (r *ViewFilterLinksResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var plan ViewFilterLinksResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.ReconcileOnApply.IsNull() && !plan.ReconcileOnApply.IsUnknown() && plan.ReconcileOnApply.ValueBool() {
		plan.ResolvedAt = types.StringUnknown()
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

func (r *ViewFilterLinksResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
	if r.client == nil {
		return
	}
	beta, err := newBetaClient(r.client.apiKey, r.client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build LaunchDarkly beta client", err.Error())
		return
	}
	r.beta = beta
}

func (r *ViewFilterLinksResource) betaClient() (*Client, error) {
	if r.beta != nil {
		return r.beta, nil
	}
	return newBetaClient(r.client.apiKey, r.client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
}

func (r *ViewFilterLinksResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ViewFilterLinksResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	betaClient, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}
	projectKey := plan.ProjectKey.ValueString()
	viewKey := plan.ViewKey.ValueString()

	exists, err := viewExists(projectKey, viewKey, betaClient)
	if err != nil {
		resp.Diagnostics.AddError("Failed to check view", err.Error())
		return
	}
	if !exists {
		resp.Diagnostics.AddError("View not found", fmt.Sprintf("cannot find view with key %q in project %q", viewKey, projectKey))
		return
	}

	flagFilter := plan.FlagFilter.ValueString()
	segmentFilter := plan.SegmentFilter.ValueString()
	segmentEnvID := plan.SegmentFilterEnvironmentID.ValueString()

	if flagFilter != "" {
		if err := linkResourcesByFilterToView(betaClient, projectKey, viewKey, FLAGS, flagFilter, ""); err != nil {
			resp.Diagnostics.AddError("Failed to link flags by filter", err.Error())
			return
		}
	}
	if segmentFilter != "" {
		if err := linkResourcesByFilterToView(betaClient, projectKey, viewKey, SEGMENTS, segmentFilter, segmentEnvID); err != nil {
			resp.Diagnostics.AddError("Failed to link segments by filter", err.Error())
			return
		}
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/%s", projectKey, viewKey))
	plan.ResolvedAt = types.StringValue(time.Now().UTC().Format(time.RFC3339))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ViewFilterLinksResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ViewFilterLinksResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	betaClient, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}
	exists, err := viewExists(data.ProjectKey.ValueString(), data.ViewKey.ValueString(), betaClient)
	if err != nil {
		resp.Diagnostics.AddError("Failed to check view", err.Error())
		return
	}
	if !exists {
		log.Printf("[WARN] view with key %q in project %q not found, removing from state", data.ViewKey.ValueString(), data.ProjectKey.ValueString())
		resp.State.RemoveResource(ctx)
		return
	}
	// Filter strings are stored in state as-is — no API resolution.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ViewFilterLinksResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ViewFilterLinksResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	betaClient, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}
	projectKey := plan.ProjectKey.ValueString()
	viewKey := plan.ViewKey.ValueString()
	segmentEnvID := plan.SegmentFilterEnvironmentID.ValueString()

	hasFlagFilterChange := !plan.FlagFilter.Equal(state.FlagFilter)
	hasSegmentFilterChange := !plan.SegmentFilter.Equal(state.SegmentFilter) || !plan.SegmentFilterEnvironmentID.Equal(state.SegmentFilterEnvironmentID)
	reconcile := plan.ReconcileOnApply.ValueBool()
	isPeriodicReconcile := reconcile && !hasFlagFilterChange && !hasSegmentFilterChange
	resyncFlags := hasFlagFilterChange || isPeriodicReconcile
	resyncSegments := hasSegmentFilterChange || isPeriodicReconcile

	if resyncFlags {
		newFilter := plan.FlagFilter.ValueString()
		oldFilter := state.FlagFilter.ValueString()
		if newFilter != "" || oldFilter != "" {
			linkedFlags, err := getLinkedResources(betaClient, projectKey, viewKey, FLAGS)
			if err != nil {
				resp.Diagnostics.AddError("Failed to get linked flags", err.Error())
				return
			}
			if len(linkedFlags) > 0 {
				keys := make([]string, len(linkedFlags))
				for i, f := range linkedFlags {
					keys[i] = f.ResourceKey
				}
				if err := unlinkResourcesFromView(betaClient, projectKey, viewKey, FLAGS, keys); err != nil {
					resp.Diagnostics.AddError("Failed to unlink flags", err.Error())
					return
				}
			}
			if newFilter != "" {
				if err := linkResourcesByFilterToView(betaClient, projectKey, viewKey, FLAGS, newFilter, ""); err != nil {
					resp.Diagnostics.AddError("Failed to link flags by filter", err.Error())
					return
				}
			}
		}
	}

	if resyncSegments {
		newFilter := plan.SegmentFilter.ValueString()
		oldFilter := state.SegmentFilter.ValueString()
		if newFilter != "" || oldFilter != "" {
			linkedSegments, err := getLinkedResources(betaClient, projectKey, viewKey, SEGMENTS)
			if err != nil {
				resp.Diagnostics.AddError("Failed to get linked segments", err.Error())
				return
			}
			if len(linkedSegments) > 0 {
				segs := make([]ViewSegmentIdentifier, len(linkedSegments))
				for i, s := range linkedSegments {
					segs[i] = ViewSegmentIdentifier{EnvironmentId: s.EnvironmentId, SegmentKey: s.ResourceKey}
				}
				if err := unlinkSegmentsFromView(betaClient, projectKey, viewKey, segs); err != nil {
					resp.Diagnostics.AddError("Failed to unlink segments", err.Error())
					return
				}
			}
			if newFilter != "" {
				if err := linkResourcesByFilterToView(betaClient, projectKey, viewKey, SEGMENTS, newFilter, segmentEnvID); err != nil {
					resp.Diagnostics.AddError("Failed to link segments by filter", err.Error())
					return
				}
			}
		}
	}

	plan.ResolvedAt = types.StringValue(time.Now().UTC().Format(time.RFC3339))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ViewFilterLinksResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ViewFilterLinksResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	betaClient, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}
	projectKey := data.ProjectKey.ValueString()
	viewKey := data.ViewKey.ValueString()

	if !data.FlagFilter.IsNull() && !data.FlagFilter.IsUnknown() && data.FlagFilter.ValueString() != "" {
		linkedFlags, err := getLinkedResources(betaClient, projectKey, viewKey, FLAGS)
		if err != nil {
			resp.Diagnostics.AddError("Failed to get linked flags", err.Error())
			return
		}
		if len(linkedFlags) > 0 {
			keys := make([]string, len(linkedFlags))
			for i, f := range linkedFlags {
				keys[i] = f.ResourceKey
			}
			if err := unlinkResourcesFromView(betaClient, projectKey, viewKey, FLAGS, keys); err != nil {
				resp.Diagnostics.AddError("Failed to unlink flags", err.Error())
				return
			}
		}
	}
	if !data.SegmentFilter.IsNull() && !data.SegmentFilter.IsUnknown() && data.SegmentFilter.ValueString() != "" {
		linkedSegments, err := getLinkedResources(betaClient, projectKey, viewKey, SEGMENTS)
		if err != nil {
			resp.Diagnostics.AddError("Failed to get linked segments", err.Error())
			return
		}
		if len(linkedSegments) > 0 {
			segs := make([]ViewSegmentIdentifier, len(linkedSegments))
			for i, s := range linkedSegments {
				segs[i] = ViewSegmentIdentifier{EnvironmentId: s.EnvironmentId, SegmentKey: s.ResourceKey}
			}
			if err := unlinkSegmentsFromView(betaClient, projectKey, viewKey, segs); err != nil {
				resp.Diagnostics.AddError("Failed to unlink segments", err.Error())
				return
			}
		}
	}
}

func (r *ViewFilterLinksResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectKey, viewKey, err := viewIdToKeys(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), projectKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(VIEW_KEY), viewKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

const viewFilterLinksDescription = `Provides a LaunchDarkly view filter links resource for linking resources to views using filter expressions.

-> **Note:** Views are available to customers on an Enterprise LaunchDarkly plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).

~> **Beta:** This resource uses a beta API. Beta resources may change or be removed in future versions.

This resource allows you to link all flags and/or segments matching a filter expression to a specific view. The filter is resolved at apply time — the backend finds all resources matching the filter and links them to the view.

-> **Note:** Filter-based links are point-in-time. By default, filters are resolved only when this resource is created or updated (for example, when ` + "`flag_filter`" + ` changes). Set ` + "`reconcile_on_apply = true`" + ` to force re-resolution on every ` + "`terraform apply`" + `.`
