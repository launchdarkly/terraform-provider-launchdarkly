package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &ObservabilityFilterSettingsResource{}
	_ resource.ResourceWithImportState = &ObservabilityFilterSettingsResource{}
)

// observabilitySettingsImportKey marks state that was just imported, so the
// first Read adopts every product that has a rule order instead of leaving
// rule_order unmanaged.
const observabilitySettingsImportKey = "observability_filter_settings_import"

type ObservabilityFilterSettingsResource struct {
	client *Client
}

type ObservabilityFilterSettingsResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectKey       types.String `tfsdk:"project_key"`
	MinuteRateLimits types.Object `tfsdk:"minute_rate_limits"`
	RuleOrder        types.Object `tfsdk:"rule_order"`
}

var (
	observabilityRateLimitAttrTypes = productAttrTypes(types.Int64Type)
	observabilityRuleOrderAttrTypes = productAttrTypes(types.ListType{ElemType: types.StringType})
)

func productAttrTypes(t attr.Type) map[string]attr.Type {
	m := make(map[string]attr.Type, len(observabilityProductTypes))
	for _, p := range observabilityProductTypes {
		m[p] = t
	}
	return m
}

func NewObservabilityFilterSettingsResource() resource.Resource {
	return &ObservabilityFilterSettingsResource{}
}

func (r *ObservabilityFilterSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_observability_filter_settings"
}

func (r *ObservabilityFilterSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	limitAttrs := map[string]schema.Attribute{}
	orderAttrs := map[string]schema.Attribute{}
	for _, p := range observabilityProductTypes {
		limitAttrs[p] = schema.Int64Attribute{
			Optional:    true,
			Description: fmt.Sprintf("The maximum number of %s ingested per minute. Omit for no limit.", p),
			Validators:  []validator.Int64{int64validator.AtLeast(0)},
		}
		orderAttrs[p] = schema.ListAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: fmt.Sprintf("The keys of the %s filter rules to run, in evaluation order. When set, this list is authoritative: any other %s rule, including one created in the LaunchDarkly UI, is removed from the order and stops running. Omit to leave the %s order unmanaged.", p, p, p),
			Validators: []validator.List{
				listvalidator.UniqueValues(),
				listvalidator.ValueStringsAre(observabilityRuleKeyValidator()),
			},
		}
	}

	resp.Schema = schema.Schema{
		Description: `Provides a LaunchDarkly observability filter settings resource.

` + observabilityBetaNote + `

This resource manages the observability ingestion settings of a LaunchDarkly project: per-product rate limits and the order in which ` + "`launchdarkly_observability_filter_rule`" + ` rules run.

Each project has exactly one filter settings object. Destroying this resource removes all rate limits and leaves rule orders unchanged; rules leave the order when they are deleted.

-> **Note:** A product listed in ` + "`rule_order`" + ` is managed authoritatively. A rule of that product that is not in the list does not run, and a rule created outside Terraform (for example in the LaunchDarkly UI) appears as drift in ` + "`terraform plan`" + ` and is removed from the order on the next apply. Reference rules by their resource (` + "`launchdarkly_observability_filter_rule.example.key`" + `) so that Terraform creates them before it sets the order.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The project key.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:      true,
				Description:   "The project key. A change in this field will force the destruction of the existing resource and the creation of a new one.",
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			MINUTE_RATE_LIMITS: schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Per-product limits on how many items LaunchDarkly ingests per minute. A product you omit, or omitting the whole attribute, means no limit.",
				Attributes:  limitAttrs,
			},
			RULE_ORDER: schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Per-product rule order, as lists of rule keys. Only the products you list are managed.",
				Attributes:  orderAttrs,
			},
		},
	}
}

func (r *ObservabilityFilterSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

// settingsPayload builds the PUT body. Every rate limit is sent (null means
// unlimited); only the products present in rule_order are sent, so the rest
// keep their server-side order.
func (r *ObservabilityFilterSettingsResource) settingsPayload(ctx context.Context, plan ObservabilityFilterSettingsResourceModel, diags *diag.Diagnostics) observabilityFilterSettings {
	payload := observabilityFilterSettings{
		MinuteRateLimits: map[string]*int64{},
		RuleOrder:        map[string][]string{},
	}
	limits := map[string]attr.Value{}
	if !plan.MinuteRateLimits.IsNull() && !plan.MinuteRateLimits.IsUnknown() {
		limits = plan.MinuteRateLimits.Attributes()
	}
	for _, p := range observabilityProductTypes {
		payload.MinuteRateLimits[p] = nil
		if v, ok := limits[p].(types.Int64); ok && !v.IsNull() && !v.IsUnknown() {
			n := v.ValueInt64()
			payload.MinuteRateLimits[p] = &n
		}
	}
	if !plan.RuleOrder.IsNull() && !plan.RuleOrder.IsUnknown() {
		for p, v := range plan.RuleOrder.Attributes() {
			l, ok := v.(types.List)
			if !ok || l.IsNull() {
				continue
			}
			keys, d := stringSliceFromList(ctx, l)
			diags.Append(d...)
			if keys == nil {
				keys = []string{}
			}
			payload.RuleOrder[p] = keys
		}
	}
	return payload
}

func (r *ObservabilityFilterSettingsResource) apply(ctx context.Context, plan *ObservabilityFilterSettingsResourceModel, diags *diag.Diagnostics, verb string) {
	projectKey := plan.ProjectKey.ValueString()
	payload := r.settingsPayload(ctx, *plan, diags)
	if diags.HasError() {
		return
	}
	settings, err := putObservabilityFilterSettings(ctx, r.client, projectKey, payload)
	if err != nil {
		addObservabilityError(diags, fmt.Sprintf("Failed to %s observability filter settings for project %q", verb, projectKey), err, "")
		return
	}
	setObservabilitySettingsModel(ctx, settings, plan, false, diags)
}

func (r *ObservabilityFilterSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ObservabilityFilterSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, &plan, &resp.Diagnostics, "create")
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ObservabilityFilterSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ObservabilityFilterSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	importing := false
	if marker, d := req.Private.GetKey(ctx, observabilitySettingsImportKey); !d.HasError() && len(marker) > 0 {
		importing = true
	}
	projectKey := data.ProjectKey.ValueString()
	settings, err := getObservabilityFilterSettings(ctx, r.client, projectKey)
	if err != nil {
		if isObservabilityNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addObservabilityError(&resp.Diagnostics, fmt.Sprintf("Failed to read observability filter settings for project %q", projectKey), err, "")
		return
	}
	setObservabilitySettingsModel(ctx, settings, &data, importing, &resp.Diagnostics)
	if importing {
		resp.Diagnostics.Append(resp.Private.SetKey(ctx, observabilitySettingsImportKey, nil)...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ObservabilityFilterSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ObservabilityFilterSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, &plan, &resp.Diagnostics, "update")
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete clears every rate limit. Rule orders are left alone: each rule
// leaves its product's order when the rule itself is deleted.
func (r *ObservabilityFilterSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ObservabilityFilterSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectKey := state.ProjectKey.ValueString()
	payload := observabilityFilterSettings{MinuteRateLimits: map[string]*int64{}, RuleOrder: map[string][]string{}}
	for _, p := range observabilityProductTypes {
		payload.MinuteRateLimits[p] = nil
	}
	if _, err := putObservabilityFilterSettings(ctx, r.client, projectKey, payload); err != nil && !isObservabilityNotFound(err) {
		addObservabilityError(&resp.Diagnostics, fmt.Sprintf("Failed to reset observability filter settings for project %q", projectKey), err, "")
	}
}

func (r *ObservabilityFilterSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Invalid import ID", "expected `project_key`")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, observabilitySettingsImportKey, []byte(`true`))...)
}

// setObservabilitySettingsModel copies server settings into data, keeping
// the user's choice of managed products: rule_order lists are filled only for
// products data already manages (non-null), except right after import, when
// every product with a non-empty order is adopted.
func setObservabilitySettingsModel(ctx context.Context, settings *observabilityFilterSettings, data *ObservabilityFilterSettingsResourceModel, importing bool, diags *diag.Diagnostics) {
	projectKey := data.ProjectKey.ValueString()
	data.ID = types.StringValue(projectKey)

	limitVals := map[string]attr.Value{}
	anyLimit := false
	for _, p := range observabilityProductTypes {
		if v := settings.MinuteRateLimits[p]; v != nil {
			limitVals[p] = types.Int64Value(*v)
			anyLimit = true
		} else {
			limitVals[p] = types.Int64Null()
		}
	}
	if !anyLimit && data.MinuteRateLimits.IsNull() {
		data.MinuteRateLimits = types.ObjectNull(observabilityRateLimitAttrTypes)
	} else {
		obj, d := types.ObjectValue(observabilityRateLimitAttrTypes, limitVals)
		diags.Append(d...)
		data.MinuteRateLimits = obj
	}

	prior := map[string]attr.Value{}
	if !data.RuleOrder.IsNull() && !data.RuleOrder.IsUnknown() {
		prior = data.RuleOrder.Attributes()
	}
	orderVals := map[string]attr.Value{}
	anyManaged := false
	for _, p := range observabilityProductTypes {
		keys := settings.RuleOrder[p]
		managed := importing && len(keys) > 0
		if l, ok := prior[p].(types.List); ok && !l.IsNull() {
			managed = true
		}
		if !managed {
			orderVals[p] = types.ListNull(types.StringType)
			continue
		}
		anyManaged = true
		if keys == nil {
			keys = []string{}
		}
		l, d := types.ListValueFrom(ctx, types.StringType, keys)
		diags.Append(d...)
		orderVals[p] = l
	}
	if !anyManaged && data.RuleOrder.IsNull() {
		data.RuleOrder = types.ObjectNull(observabilityRuleOrderAttrTypes)
		return
	}
	obj, d := types.ObjectValue(observabilityRuleOrderAttrTypes, orderVals)
	diags.Append(d...)
	data.RuleOrder = obj
}
