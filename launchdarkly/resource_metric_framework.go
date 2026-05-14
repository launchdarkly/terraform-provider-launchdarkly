package launchdarkly

import (
	"context"
	"fmt"
	"net/http"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

const CUSTOM_METRIC_DEFAULT_SUCCESS_CRITERIA = "HigherThanBaseline"

var (
	_ resource.Resource                = &MetricResource{}
	_ resource.ResourceWithImportState = &MetricResource{}
	_ resource.ResourceWithModifyPlan  = &MetricResource{}
)

type MetricResource struct {
	client *Client
}

type MetricResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	ProjectKey                types.String `tfsdk:"project_key"`
	Key                       types.String `tfsdk:"key"`
	Name                      types.String `tfsdk:"name"`
	Kind                      types.String `tfsdk:"kind"`
	MaintainerID              types.String `tfsdk:"maintainer_id"`
	Description               types.String `tfsdk:"description"`
	Tags                      types.Set    `tfsdk:"tags"`
	IsActive                  types.Bool   `tfsdk:"is_active"`
	IsNumeric                 types.Bool   `tfsdk:"is_numeric"`
	Unit                      types.String `tfsdk:"unit"`
	Selector                  types.String `tfsdk:"selector"`
	EventKey                  types.String `tfsdk:"event_key"`
	SuccessCriteria           types.String `tfsdk:"success_criteria"`
	URLs                      types.List   `tfsdk:"urls"`
	RandomizationUnits        types.Set    `tfsdk:"randomization_units"`
	IncludeUnitsWithoutEvents types.Bool   `tfsdk:"include_units_without_events"`
	UnitAggregationType       types.String `tfsdk:"unit_aggregation_type"`
	AnalysisType              types.String `tfsdk:"analysis_type"`
	PercentileValue           types.Int64  `tfsdk:"percentile_value"`
	Version                   types.Int64  `tfsdk:"version"`
}

func NewMetricResource() resource.Resource { return &MetricResource{} }

func (r *MetricResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_metric"
}

var metricUrlAttrTypes = map[string]attr.Type{
	KIND:      types.StringType,
	URL:       types.StringType,
	SUBSTRING: types.StringType,
	PATTERN:   types.StringType,
}

func (r *MetricResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly metric resource.\n\nThis resource allows you to create and manage metrics within your LaunchDarkly organization.\n\nTo learn more about metrics and experimentation, read [Experimentation Documentation](https://docs.launchdarkly.com/home/experimentation).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:      true,
				Description:   "The metrics's project key. A change in this field will force the destruction of the existing resource and the creation of a new one.",
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			KEY: schema.StringAttribute{
				Required:      true,
				Description:   "The unique key that references the metric. A change in this field will force the destruction of the existing resource and the creation of a new one.",
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			NAME: schema.StringAttribute{
				Required:    true,
				Description: "The human-friendly name for the metric.",
			},
			KIND: schema.StringAttribute{
				Required:      true,
				Description:   "The metric type. Available choices are `click`, `custom`, and `pageview`. A change in this field will force the destruction of the existing resource and the creation of a new one.",
				Validators:    []validator.String{oneOfValidator{allowed: []string{"pageview", "click", "custom"}}},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			MAINTAINER_ID: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The LaunchDarkly member ID of the member who will maintain the metric. If not set, the API will automatically apply the member associated with your Terraform API key or the most recently-set maintainer",
				Validators:  []validator.String{idValidator()},
			},
			DESCRIPTION: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The description of the metric's purpose.",
			},
			TAGS: schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Tags associated with this resource.",
			},
			IS_ACTIVE: schema.BoolAttribute{
				Optional:           true,
				Computed:           true,
				Description:        "Ignored. All metrics are considered active.",
				DeprecationMessage: "No longer in use. This field will be removed in a future major release of the LaunchDarkly provider.",
			},
			IS_NUMERIC: schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether a `custom` metric is a numeric metric or not.",
			},
			UNIT: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "(Required for kind `custom`) The unit for numeric `custom` metrics.",
			},
			SELECTOR: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The CSS selector for your metric (if click metric)",
			},
			EVENT_KEY: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The event key for your metric (if custom metric)",
			},
			SUCCESS_CRITERIA: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The success criteria for your metric (if numeric metric). Available choices are `HigherThanBaseline` and `LowerThanBaseline`.",
				Validators:  []validator.String{oneOfValidator{allowed: []string{"HigherThanBaseline", "LowerThanBaseline"}}},
			},
			RANDOMIZATION_UNITS: schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "A set of one or more context kinds that this metric can measure events from. Metrics can only use context kinds marked as \"Available for experiments.\" For more information, read [Allocating experiment audiences](https://docs.launchdarkly.com/home/creating-experiments/allocation).",
			},
			INCLUDE_UNITS_WITHOUT_EVENTS: schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Include units that did not send any events and set their value to 0.",
			},
			UNIT_AGGREGATION_TYPE: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("average"),
				Description: "The method by which multiple unit event values are aggregated. Available choices are `average` and `sum`.",
				Validators:  []validator.String{oneOfValidator{allowed: []string{"average", "sum"}}},
			},
			ANALYSIS_TYPE: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("mean"),
				Description: "The method for analyzing metric events. Available choices are `mean` and `percentile`.",
				Validators:  []validator.String{oneOfValidator{allowed: []string{"mean", "percentile"}}},
			},
			PERCENTILE_VALUE: schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "The percentile for the analysis method. An integer denoting the target percentile between 0 and 100. Required when analysis_type is percentile.",
			},
			VERSION: schema.Int64Attribute{
				Computed:      true,
				Description:   "Version of the metric",
				PlanModifiers: []planmodifier.Int64{
					// Recomputed on any change. ModifyPlan flips this to unknown.
				},
			},
			URLS: schema.ListNestedAttribute{
				Optional:    true,
				Description: "List of URLs that you want to associate with the metric.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						KIND: schema.StringAttribute{
							Required:    true,
							Description: "The URL type. Available choices are `exact`, `canonical`, `substring` and `regex`.",
							Validators:  []validator.String{oneOfValidator{allowed: []string{"exact", "canonical", "substring", "regex"}}},
						},
						URL: schema.StringAttribute{
							Optional:    true,
							Description: "(Required for kind `exact` and `canonical`) The exact or canonical URL.",
						},
						SUBSTRING: schema.StringAttribute{
							Optional:    true,
							Description: "(Required for kind `substring`) The URL substring to match by.",
						},
						PATTERN: schema.StringAttribute{
							Optional:    true,
							Description: "(Required for kind `regex`) The regex pattern to match by.",
						},
					},
				},
			},
		},
	}
}

// ModifyPlan ports the SDKv2 customizeMetricDiff. Validates per-kind
// required/forbidden fields, defaults success_criteria for custom
// metrics, gates percentile_value on analysis_type=percentile, and
// marks `version` as unknown when any other attribute changes.
func (r *MetricResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan MetricResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	kind := plan.Kind.ValueString()
	selectorSet := !plan.Selector.IsNull() && !plan.Selector.IsUnknown() && plan.Selector.ValueString() != ""
	unitSet := !plan.Unit.IsNull() && !plan.Unit.IsUnknown() && plan.Unit.ValueString() != ""
	eventKeySet := !plan.EventKey.IsNull() && !plan.EventKey.IsUnknown() && plan.EventKey.ValueString() != ""
	successCriteriaSet := !plan.SuccessCriteria.IsNull() && !plan.SuccessCriteria.IsUnknown() && plan.SuccessCriteria.ValueString() != ""
	urlsCount := 0
	if !plan.URLs.IsNull() && !plan.URLs.IsUnknown() {
		urlsCount = len(plan.URLs.Elements())
	}
	percentileSet := !plan.PercentileValue.IsNull() && !plan.PercentileValue.IsUnknown() && plan.PercentileValue.ValueInt64() != 0
	analysisType := plan.AnalysisType.ValueString()
	includeUnitsSet := !plan.IncludeUnitsWithoutEvents.IsNull() && !plan.IncludeUnitsWithoutEvents.IsUnknown()

	switch kind {
	case "click":
		if !selectorSet {
			resp.Diagnostics.AddError("click metrics require 'selector' to be set", "")
		}
		if urlsCount == 0 {
			resp.Diagnostics.AddError("click metrics require an 'urls' block to be set", "")
		}
		if successCriteriaSet {
			resp.Diagnostics.AddError("click metrics do not accept 'success_criteria'", "")
		}
		if unitSet {
			resp.Diagnostics.AddError("click metrics do not accept 'unit'", "")
		}
		if eventKeySet {
			resp.Diagnostics.AddError("click metrics do not accept 'event_key'", "")
		}
	case "custom":
		// Default success_criteria when missing.
		if !successCriteriaSet {
			plan.SuccessCriteria = types.StringValue(CUSTOM_METRIC_DEFAULT_SUCCESS_CRITERIA)
		}
		isNumeric := plan.IsNumeric.ValueBool()
		if isNumeric {
			if plan.SuccessCriteria.ValueString() == "" {
				resp.Diagnostics.AddError("numeric custom metrics require 'success_criteria' to be set", "")
			}
			if !unitSet {
				resp.Diagnostics.AddError("numeric custom metrics require 'unit' to be set", "")
			}
		}
		if !eventKeySet {
			resp.Diagnostics.AddError("custom metrics require 'event_key' to be set", "")
		}
		if urlsCount > 0 {
			resp.Diagnostics.AddError("custom metrics do not accept a 'urls' block", "")
		}
		if selectorSet {
			resp.Diagnostics.AddError("custom metrics do not accept 'selector'", "")
		}
	case "pageview":
		if urlsCount == 0 {
			resp.Diagnostics.AddError("pageview metrics require an 'urls' block to be set", "")
		}
		if successCriteriaSet {
			resp.Diagnostics.AddError("pageview metrics do not accept 'success_criteria'", "")
		}
		if unitSet {
			resp.Diagnostics.AddError("pageview metrics do not accept 'unit'", "")
		}
		if eventKeySet {
			resp.Diagnostics.AddError("pageview metrics do not accept 'event_key'", "")
		}
		if selectorSet {
			resp.Diagnostics.AddError("pageview metrics do not accept 'selector'", "")
		}
	}

	// URLs sub-kind validation
	if urlsCount > 0 {
		var urls []metricURLModel
		resp.Diagnostics.Append(plan.URLs.ElementsAs(ctx, &urls, false)...)
		for _, u := range urls {
			if !metricURLEntryValid(u) {
				resp.Diagnostics.AddError("'urls' block is misconfigured, please check documentation for required fields", "")
				break
			}
		}
	}

	if analysisType == "percentile" {
		if !percentileSet {
			resp.Diagnostics.AddError("percentile_value is required when analysis_type is percentile", "")
		}
		if includeUnitsSet && plan.IncludeUnitsWithoutEvents.ValueBool() {
			resp.Diagnostics.AddError("include_units_without_events is not supported for percentile metrics", "")
		}
	} else if percentileSet {
		resp.Diagnostics.AddError(fmt.Sprintf("%s type metrics can not have percentile values", analysisType), "")
	}

	if !includeUnitsSet {
		// Default: false for percentile, true otherwise (matches SDKv2).
		if analysisType == "percentile" {
			plan.IncludeUnitsWithoutEvents = types.BoolValue(false)
		} else {
			plan.IncludeUnitsWithoutEvents = types.BoolValue(true)
		}
	}

	// Mirror SDKv2's diff.SetNewComputed(VERSION) — only mark version
	// Unknown when something *other than* version actually changed.
	if !req.State.Raw.IsNull() {
		var state MetricResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		// Compare without version: temporarily mirror state.Version
		// into plan, check equality, restore.
		planVersion := plan.Version
		plan.Version = state.Version
		if !reflect.DeepEqual(plan, state) {
			plan.Version = types.Int64Unknown()
		} else {
			plan.Version = planVersion
		}
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

type metricURLModel struct {
	Kind      types.String `tfsdk:"kind"`
	URL       types.String `tfsdk:"url"`
	Substring types.String `tfsdk:"substring"`
	Pattern   types.String `tfsdk:"pattern"`
}

func metricURLEntryValid(u metricURLModel) bool {
	kind := u.Kind.ValueString()
	url := u.URL.ValueString()
	substring := u.Substring.ValueString()
	pattern := u.Pattern.ValueString()
	switch kind {
	case "exact", "canonical":
		if url == "" {
			return false
		}
		if pattern != "" || substring != "" {
			return false
		}
	case "substring":
		if substring == "" {
			return false
		}
		if pattern != "" || url != "" {
			return false
		}
	case "regex":
		if pattern == "" {
			return false
		}
		if substring != "" || url != "" {
			return false
		}
	}
	return true
}

func (r *MetricResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *MetricResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MetricResourceModel
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

	tags, d := stringSliceFromSet(ctx, plan.Tags)
	resp.Diagnostics.Append(d...)
	randomization, d := stringSliceFromSet(ctx, plan.RandomizationUnits)
	resp.Diagnostics.Append(d...)

	var urls []metricURLModel
	if !plan.URLs.IsNull() && !plan.URLs.IsUnknown() {
		resp.Diagnostics.Append(plan.URLs.ElementsAs(ctx, &urls, false)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	key := plan.Key.ValueString()
	name := plan.Name.ValueString()
	kind := plan.Kind.ValueString()
	description := plan.Description.ValueString()
	isNumeric := plan.IsNumeric.ValueBool()
	unit := plan.Unit.ValueString()
	selector := plan.Selector.ValueString()
	eventKey := plan.EventKey.ValueString()
	unitAggType := plan.UnitAggregationType.ValueString()
	analysisType := plan.AnalysisType.ValueString()
	includeUnits := plan.IncludeUnitsWithoutEvents.ValueBool()
	eventDefaultDisabled := !includeUnits

	post := ldapi.MetricPost{
		Name:                &name,
		Key:                 key,
		Description:         &description,
		Tags:                tags,
		Kind:                kind,
		IsNumeric:           &isNumeric,
		Selector:            &selector,
		Urls:                metricURLPostsFromModels(urls),
		RandomizationUnits:  randomization,
		Unit:                &unit,
		EventKey:            &eventKey,
		UnitAggregationType: &unitAggType,
		AnalysisType:        &analysisType,
		EventDefault:        &ldapi.MetricEventDefaultRep{Disabled: &eventDefaultDisabled},
	}

	if pv := plan.PercentileValue.ValueInt64(); pv != 0 {
		pvI32 := int32(pv)
		post.PercentileValue = &pvI32
	}
	if sc := plan.SuccessCriteria.ValueString(); sc != "" {
		post.SuccessCriteria = &sc
	} else if kind == "custom" {
		def := CUSTOM_METRIC_DEFAULT_SUCCESS_CRITERIA
		post.SuccessCriteria = &def
	}

	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.MetricsApi.PostMetric(r.client.ctx, projectKey).MetricPost(post).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error creating metric resource: %q", key), err)
		return
	}

	// Maintainer can only be set after create (SDKv2 parity).
	if !plan.MaintainerID.IsNull() && !plan.MaintainerID.IsUnknown() && plan.MaintainerID.ValueString() != "" {
		if err := r.applyMetricUpdate(ctx, &plan, false); err != nil {
			addLdapiError(&resp.Diagnostics, "Error setting maintainer on new metric", err)
			// Best-effort cleanup
			_ = r.client.withConcurrency(r.client.ctx, func() error {
				_, e := r.client.ld.MetricsApi.DeleteMetric(r.client.ctx, projectKey, key).Execute()
				return e
			})
			return
		}
	}

	plan.ID = types.StringValue(projectKey + "/" + key)
	r.readIntoModel(ctx, projectKey, key, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MetricResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MetricResourceModel
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

func (r *MetricResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan MetricResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyMetricUpdate(ctx, &plan, true); err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error updating metric resource %q from project %q", plan.Key.ValueString(), plan.ProjectKey.ValueString()), err)
		return
	}

	r.readIntoModel(ctx, plan.ProjectKey.ValueString(), plan.Key.ValueString(), &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MetricResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MetricResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, e := r.client.ld.MetricsApi.DeleteMetric(r.client.ctx, data.ProjectKey.ValueString(), data.Key.ValueString()).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error deleting metric resource %q", data.Key.ValueString()), err)
	}
}

func (r *MetricResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectKey, metricKey, err := metricIdToKeys(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), projectKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(KEY), metricKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// applyMetricUpdate issues the PATCH that the SDKv2 update did. Shared
// between Update and the post-create maintainer set.
func (r *MetricResource) applyMetricUpdate(ctx context.Context, plan *MetricResourceModel, includeRandomizationUnits bool) error {
	projectKey := plan.ProjectKey.ValueString()
	key := plan.Key.ValueString()

	tags, _ := stringSliceFromSet(ctx, plan.Tags)
	var urls []metricURLModel
	if !plan.URLs.IsNull() && !plan.URLs.IsUnknown() {
		_ = plan.URLs.ElementsAs(ctx, &urls, false)
	}
	urlPosts := metricURLPostsFromModels(urls)

	name := plan.Name.ValueString()
	description := plan.Description.ValueString()
	kind := plan.Kind.ValueString()
	isActive := plan.IsActive.ValueBool()
	isNumeric := plan.IsNumeric.ValueBool()
	unit := plan.Unit.ValueString()
	selector := plan.Selector.ValueString()
	eventKey := plan.EventKey.ValueString()
	unitAggType := plan.UnitAggregationType.ValueString()
	analysisType := plan.AnalysisType.ValueString()
	includeUnits := plan.IncludeUnitsWithoutEvents.ValueBool()

	patch := []ldapi.PatchOperation{
		patchReplace("/name", name),
		patchReplace("/description", description),
		patchReplace("/tags", tags),
		patchReplace("/kind", kind),
		patchReplace("/isActive", isActive),
		patchReplace("/isNumeric", isNumeric),
		patchReplace("/urls", urlPosts),
		patchReplace("/unit", unit),
		patchReplace("/selector", selector),
		patchReplace("/eventKey", eventKey),
		patchReplace("/unitAggregationType", unitAggType),
		patchReplace("/analysisType", analysisType),
		patchReplace("/eventDefault/disabled", !includeUnits),
	}

	if pv := plan.PercentileValue.ValueInt64(); pv != 0 {
		patch = append(patch, patchReplace("/percentileValue", int32(pv)))
	} else {
		patch = append(patch, patchReplace("/percentileValue", nil))
	}

	if sc := plan.SuccessCriteria.ValueString(); sc != "" {
		patch = append(patch, patchReplace("/successCriteria", sc))
	} else if kind == "custom" {
		patch = append(patch, patchReplace("/successCriteria", CUSTOM_METRIC_DEFAULT_SUCCESS_CRITERIA))
	}

	if mid := plan.MaintainerID.ValueString(); mid != "" {
		patch = append(patch, patchReplace("/maintainerId", mid))
	}

	if includeRandomizationUnits {
		ru, _ := stringSliceFromSet(ctx, plan.RandomizationUnits)
		if len(ru) > 0 {
			patch = append(patch, patchReplace("/randomizationUnits", ru))
		}
	}

	return r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.MetricsApi.PatchMetric(r.client.ctx, projectKey, key).PatchOperation(patch).Execute()
		return e
	})
}

func (r *MetricResource) readIntoModel(
	ctx context.Context,
	projectKey, key string,
	data *MetricResourceModel,
	diags *diag.Diagnostics,
) {
	var metric *ldapi.MetricRep
	var res *http.Response
	var err error
	err = r.client.withConcurrency(r.client.ctx, func() error {
		metric, res, err = r.client.ld.MetricsApi.GetMetric(r.client.ctx, projectKey, key).Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError(fmt.Sprintf("Failed to get metric %q", key), handleLdapiErr(err).Error())
		return
	}

	data.ID = types.StringValue(projectKey + "/" + key)
	data.ProjectKey = types.StringValue(projectKey)
	data.Key = types.StringValue(metric.Key)
	data.Name = types.StringValue(metric.Name)
	data.Description = stringValueFromPointer(metric.Description)
	data.Kind = types.StringValue(metric.Kind)
	if metric.IsActive != nil {
		data.IsActive = types.BoolValue(*metric.IsActive)
	} else {
		data.IsActive = types.BoolValue(false)
	}
	if metric.IsNumeric != nil {
		data.IsNumeric = types.BoolValue(*metric.IsNumeric)
	} else {
		data.IsNumeric = types.BoolValue(false)
	}
	data.Selector = stringValueFromPointer(metric.Selector)
	data.Unit = stringValueFromPointer(metric.Unit)
	data.EventKey = stringValueFromPointer(metric.EventKey)
	data.SuccessCriteria = stringValueFromPointer(metric.SuccessCriteria)
	if metric.EventDefault != nil && metric.EventDefault.Disabled != nil {
		data.IncludeUnitsWithoutEvents = types.BoolValue(!*metric.EventDefault.Disabled)
	}
	if metric.UnitAggregationType != nil {
		data.UnitAggregationType = types.StringValue(*metric.UnitAggregationType)
	}
	if metric.AnalysisType != nil {
		data.AnalysisType = types.StringValue(*metric.AnalysisType)
	}
	if metric.PercentileValue != nil {
		data.PercentileValue = types.Int64Value(int64(*metric.PercentileValue))
	} else {
		data.PercentileValue = types.Int64Value(0)
	}
	if metric.Version != nil {
		data.Version = types.Int64Value(int64(*metric.Version))
	} else {
		data.Version = types.Int64Value(0)
	}

	if maintainer := metric.Maintainer; maintainer != nil {
		if id := maintainer.GetId(); id != "" {
			data.MaintainerID = types.StringValue(id)
		}
	}
	if data.MaintainerID.IsNull() || data.MaintainerID.IsUnknown() {
		data.MaintainerID = types.StringValue("")
	}

	tagsSet, d := setFromStringSlice(ctx, metric.Tags)
	diags.Append(d...)
	data.Tags = tagsSet

	ruSet, d := setFromStringSlice(ctx, metric.RandomizationUnits)
	diags.Append(d...)
	data.RandomizationUnits = ruSet

	urlObjType := types.ObjectType{AttrTypes: metricUrlAttrTypes}
	if len(metric.Urls) == 0 {
		data.URLs = types.ListNull(urlObjType)
	} else {
		urlElems := make([]attr.Value, 0, len(metric.Urls))
		for _, u := range metric.Urls {
			obj, d := types.ObjectValue(metricUrlAttrTypes, map[string]attr.Value{
				KIND:      stringFromMap(u, "kind"),
				URL:       stringFromMap(u, "url"),
				SUBSTRING: stringFromMap(u, "substring"),
				PATTERN:   stringFromMap(u, "pattern"),
			})
			diags.Append(d...)
			urlElems = append(urlElems, obj)
		}
		urlList, d := types.ListValue(urlObjType, urlElems)
		diags.Append(d...)
		data.URLs = urlList
	}
}

func stringFromMap(m map[string]interface{}, key string) types.String {
	v, ok := m[key]
	if !ok || v == nil {
		return types.StringNull()
	}
	s, ok := v.(string)
	if !ok {
		return types.StringNull()
	}
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func metricURLPostsFromModels(in []metricURLModel) []ldapi.UrlPost {
	out := make([]ldapi.UrlPost, len(in))
	for i, u := range in {
		k := u.Kind.ValueString()
		url := u.URL.ValueString()
		sub := u.Substring.ValueString()
		pat := u.Pattern.ValueString()
		post := ldapi.UrlPost{Kind: &k}
		if url != "" {
			post.Url = &url
		}
		if sub != "" {
			post.Substring = &sub
		}
		if pat != "" {
			post.Pattern = &pat
		}
		out[i] = post
	}
	return out
}
