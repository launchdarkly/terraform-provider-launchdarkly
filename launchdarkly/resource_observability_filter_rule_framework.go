package launchdarkly

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &ObservabilityFilterRuleResource{}
	_ resource.ResourceWithImportState    = &ObservabilityFilterRuleResource{}
	_ resource.ResourceWithValidateConfig = &ObservabilityFilterRuleResource{}
)

const observabilityBetaNote = "~> **Beta:** This resource uses a beta API. Beta resources may change or be removed in future versions."

// observabilityCredentialNote: the endpoints forward the caller's own credential to the observability backend.
const observabilityCredentialNote = "-> **Note:** Observability filter settings require a personal API access token, or an OAuth token with the `observability` scope. Service tokens are not supported."

type ObservabilityFilterRuleResource struct {
	client *Client
}

type ObservabilityFilterRuleResourceModel struct {
	ID          types.String  `tfsdk:"id"`
	ProjectKey  types.String  `tfsdk:"project_key"`
	Key         types.String  `tfsdk:"key"`
	ProductType types.String  `tfsdk:"product_type"`
	Name        types.String  `tfsdk:"name"`
	Enabled     types.Bool    `tfsdk:"enabled"`
	Query       types.String  `tfsdk:"query"`
	Operation   types.String  `tfsdk:"operation"`
	Rate        types.Float64 `tfsdk:"rate"`
	Attributes  types.Set     `tfsdk:"attributes"`
	Version     types.Int64   `tfsdk:"version"`
}

func NewObservabilityFilterRuleResource() resource.Resource {
	return &ObservabilityFilterRuleResource{}
}

func (r *ObservabilityFilterRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_observability_filter_rule"
}

func (r *ObservabilityFilterRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	forceNew := " A change in this field will force the destruction of the existing resource and the creation of a new one."
	resp.Schema = schema.Schema{
		Description: `Provides a LaunchDarkly observability ingestion filter rule resource.

` + observabilityBetaNote + `

` + observabilityCredentialNote + `

This resource allows you to create and manage a filter rule that includes or excludes sessions, errors, logs, traces, or metrics at ingestion for a LaunchDarkly project. LaunchDarkly adds a new rule to the end of its product's rule order. To control the order, or to make Terraform the authority on which rules of a product run, use ` + "`launchdarkly_observability_filter_settings`" + `.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The rule ID, in the format `project_key/key`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:      true,
				Description:   "The project key." + forceNew,
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			KEY: schema.StringAttribute{
				Required:      true,
				Description:   "The rule's unique key within the project. Must be 1-256 letters, numbers, `.`, `-`, or `_`, starting with a letter or number." + forceNew,
				Validators:    []validator.String{observabilityRuleKeyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			PRODUCT_TYPE: schema.StringAttribute{
				Required:      true,
				Description:   "The product the rule filters. Must be one of `sessions`, `errors`, `logs`, `traces`, or `metrics`." + forceNew,
				Validators:    []validator.String{stringvalidator.OneOf(observabilityProductTypes...)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			NAME: schema.StringAttribute{
				Required:    true,
				Description: "The rule's human-readable name.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			ENABLED: schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the rule is enabled. Defaults to `true`.",
			},
			QUERY: schema.StringAttribute{
				Required:    true,
				Description: "The search query that selects the data the rule applies to, in LaunchDarkly observability search syntax, for example `service_name=api route=/healthz`.",
			},
			OPERATION: schema.StringAttribute{
				Required:    true,
				Description: "What to do with matching data. Must be `include` or `exclude`.",
				Validators:  []validator.String{stringvalidator.OneOf(observabilityOperations...)},
			},
			RATE: schema.Float64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     float64default.StaticFloat64(1),
				Description: "The fraction of matching data the operation applies to, from `0` to `1`. Defaults to `1`. Rules for `metrics` must use `1`.",
				Validators:  []validator.Float64{float64validator.Between(0, 1)},
			},
			ATTRIBUTES: schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Attribute keys to filter instead of the whole item. With `exclude`, LaunchDarkly removes these attributes; with `include`, it keeps only these attributes. Only valid for `metrics` rules.",
				Validators:  []validator.Set{setvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1))},
			},
			VERSION: schema.Int64Attribute{
				Computed:    true,
				Description: "The rule's version. LaunchDarkly increments it on every change; Terraform sends it on update so that a concurrent change is rejected rather than overwritten.",
			},
		},
	}
}

func (r *ObservabilityFilterRuleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ObservabilityFilterRuleResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() || data.ProductType.IsNull() || data.ProductType.IsUnknown() {
		return
	}
	isMetrics := data.ProductType.ValueString() == METRICS
	if isMetrics && !data.Rate.IsNull() && !data.Rate.IsUnknown() && data.Rate.ValueFloat64() != 1 {
		resp.Diagnostics.AddAttributeError(path.Root(RATE), "Invalid rate for a metrics rule",
			fmt.Sprintf("Rules with `product_type = \"metrics\"` must use `rate = 1`, got %v.", data.Rate.ValueFloat64()))
	}
	if !isMetrics && !data.Attributes.IsNull() && !data.Attributes.IsUnknown() && len(data.Attributes.Elements()) > 0 {
		resp.Diagnostics.AddAttributeError(path.Root(ATTRIBUTES), "Attributes are only supported for metrics rules",
			fmt.Sprintf("`attributes` can only be set when `product_type = \"metrics\"`, got %q.", data.ProductType.ValueString()))
	}
}

func (r *ObservabilityFilterRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func observabilityRuleAttributesFromPlan(ctx context.Context, plan ObservabilityFilterRuleResourceModel, diags *diag.Diagnostics) []string {
	attrs := []string{}
	if plan.Attributes.IsNull() || plan.Attributes.IsUnknown() {
		return attrs
	}
	vals, d := stringSliceFromSet(ctx, plan.Attributes)
	diags.Append(d...)
	if vals != nil {
		attrs = vals
	}
	return attrs
}

func (r *ObservabilityFilterRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ObservabilityFilterRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectKey, key := plan.ProjectKey.ValueString(), plan.Key.ValueString()
	post := observabilityFilterRulePost{
		Key:         key,
		Name:        plan.Name.ValueString(),
		ProductType: plan.ProductType.ValueString(),
		Enabled:     plan.Enabled.ValueBool(),
		Query:       plan.Query.ValueString(),
		Operation:   plan.Operation.ValueString(),
		Rate:        plan.Rate.ValueFloat64(),
		Attributes:  observabilityRuleAttributesFromPlan(ctx, plan, &resp.Diagnostics),
	}
	if resp.Diagnostics.HasError() {
		return
	}
	rule, err := createObservabilityFilterRule(ctx, r.client, projectKey, post)
	if err != nil {
		addObservabilityError(&resp.Diagnostics, fmt.Sprintf("Failed to create observability filter rule %q in project %q", key, projectKey), err,
			fmt.Sprintf("A filter rule with key %q already exists in project %q. Choose another key, or bring the existing rule under Terraform with `terraform import <address> %s/%s`.", key, projectKey, projectKey, key))
		return
	}
	r.setModelFromRule(ctx, rule, projectKey, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ObservabilityFilterRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ObservabilityFilterRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectKey, key := data.ProjectKey.ValueString(), data.Key.ValueString()
	rule, err := getObservabilityFilterRule(ctx, r.client, projectKey, key)
	if err != nil {
		if isObservabilityNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addObservabilityError(&resp.Diagnostics, fmt.Sprintf("Failed to read observability filter rule %q in project %q", key, projectKey), err, "")
		return
	}
	r.setModelFromRule(ctx, rule, projectKey, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ObservabilityFilterRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ObservabilityFilterRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectKey, key := plan.ProjectKey.ValueString(), plan.Key.ValueString()
	put := observabilityFilterRulePut{
		Name:       plan.Name.ValueString(),
		Enabled:    plan.Enabled.ValueBool(),
		Query:      plan.Query.ValueString(),
		Operation:  plan.Operation.ValueString(),
		Rate:       plan.Rate.ValueFloat64(),
		Attributes: observabilityRuleAttributesFromPlan(ctx, plan, &resp.Diagnostics),
	}
	if !state.Version.IsNull() && !state.Version.IsUnknown() {
		v := state.Version.ValueInt64()
		put.Version = &v
	}
	if resp.Diagnostics.HasError() {
		return
	}
	rule, err := updateObservabilityFilterRule(ctx, r.client, projectKey, key, put)
	if err != nil {
		addObservabilityError(&resp.Diagnostics, fmt.Sprintf("Failed to update observability filter rule %q in project %q", key, projectKey), err, "")
		return
	}
	r.setModelFromRule(ctx, rule, projectKey, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ObservabilityFilterRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ObservabilityFilterRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectKey, key := state.ProjectKey.ValueString(), state.Key.ValueString()
	if err := deleteObservabilityFilterRule(ctx, r.client, projectKey, key); err != nil && !isObservabilityNotFound(err) {
		addObservabilityError(&resp.Diagnostics, fmt.Sprintf("Failed to delete observability filter rule %q in project %q", key, projectKey), err, "")
	}
}

func (r *ObservabilityFilterRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("expected `project_key/key`, got %q", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(KEY), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *ObservabilityFilterRuleResource) setModelFromRule(ctx context.Context, rule *observabilityFilterRule, projectKey string, data *ObservabilityFilterRuleResourceModel, diags *diag.Diagnostics) {
	data.ID = types.StringValue(projectKey + "/" + rule.Key)
	data.ProjectKey = types.StringValue(projectKey)
	data.Key = types.StringValue(rule.Key)
	data.ProductType = types.StringValue(rule.ProductType)
	data.Name = types.StringValue(rule.Name)
	data.Enabled = types.BoolValue(rule.Enabled)
	data.Query = types.StringValue(rule.Query)
	data.Operation = types.StringValue(rule.Operation)
	data.Rate = types.Float64Value(rule.Rate)
	data.Version = types.Int64Value(rule.Version)
	attrs, d := setFromStringSlicePreservingPlan(ctx, rule.Attributes, data.Attributes)
	diags.Append(d...)
	data.Attributes = attrs
}
