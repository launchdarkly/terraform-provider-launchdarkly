package launchdarkly

// Phase 3.3 scaffold for launchdarkly_metric. CRUD is a TODO marker;
// resource not registered on framework yet.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &MetricResource{}

type MetricResource struct{ client *Client }

type MetricResourceModel struct {
	ID                        types.String  `tfsdk:"id"`
	ProjectKey                types.String  `tfsdk:"project_key"`
	Key                       types.String  `tfsdk:"key"`
	Name                      types.String  `tfsdk:"name"`
	Kind                      types.String  `tfsdk:"kind"`
	MaintainerID              types.String  `tfsdk:"maintainer_id"`
	Description               types.String  `tfsdk:"description"`
	Tags                      types.Set     `tfsdk:"tags"`
	IsActive                  types.Bool    `tfsdk:"is_active"`
	IsNumeric                 types.Bool    `tfsdk:"is_numeric"`
	Unit                      types.String  `tfsdk:"unit"`
	Selector                  types.String  `tfsdk:"selector"`
	EventKey                  types.String  `tfsdk:"event_key"`
	SuccessCriteria           types.String  `tfsdk:"success_criteria"`
	URLs                      types.List    `tfsdk:"urls"`
	RandomizationUnits        types.Set     `tfsdk:"randomization_units"`
	IncludeUnitsWithoutEvents types.Bool    `tfsdk:"include_units_without_events"`
	UnitAggregationType       types.String  `tfsdk:"unit_aggregation_type"`
	AnalysisType              types.String  `tfsdk:"analysis_type"`
	PercentileValue           types.Int64   `tfsdk:"percentile_value"`
	Version                   types.Int64   `tfsdk:"version"`
	_                         types.Float64 // placeholder
}

func NewMetricResource() resource.Resource { return &MetricResource{} }

func (r *MetricResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_metric"
}

func (r *MetricResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly metric resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			PROJECT_KEY: schema.StringAttribute{
				Required:      true,
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			KEY: schema.StringAttribute{
				Required:      true,
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			NAME: schema.StringAttribute{Required: true},
			KIND: schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					oneOfValidator{allowed: []string{"pageview", "click", "custom"}},
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			MAINTAINER_ID: schema.StringAttribute{Optional: true, Computed: true},
			DESCRIPTION:   schema.StringAttribute{Optional: true, Computed: true},
			TAGS:          schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType},
			IS_ACTIVE: schema.BoolAttribute{
				Optional: true, Computed: true,
				DeprecationMessage: "No longer in use. This field will be removed in a future major release of the LaunchDarkly provider.",
			},
			IS_NUMERIC:                   schema.BoolAttribute{Optional: true, Computed: true},
			UNIT:                         schema.StringAttribute{Optional: true, Computed: true},
			SELECTOR:                     schema.StringAttribute{Optional: true, Computed: true},
			EVENT_KEY:                    schema.StringAttribute{Optional: true, Computed: true},
			SUCCESS_CRITERIA:             schema.StringAttribute{Optional: true, Computed: true},
			RANDOMIZATION_UNITS:          schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType},
			INCLUDE_UNITS_WITHOUT_EVENTS: schema.BoolAttribute{Optional: true, Computed: true},
			UNIT_AGGREGATION_TYPE:        schema.StringAttribute{Optional: true, Computed: true},
			ANALYSIS_TYPE:                schema.StringAttribute{Optional: true, Computed: true},
			PERCENTILE_VALUE:             schema.Int64Attribute{Optional: true, Computed: true},
			VERSION:                      schema.Int64Attribute{Computed: true},
		},
		Blocks: map[string]schema.Block{
			URLS: schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						KIND:      schema.StringAttribute{Required: true},
						URL:       schema.StringAttribute{Optional: true},
						SUBSTRING: schema.StringAttribute{Optional: true},
						PATTERN:   schema.StringAttribute{Optional: true},
					},
				},
			},
		},
	}
}

func (r *MetricResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *MetricResource) Create(_ context.Context, _ resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError("launchdarkly_metric scaffold", "Phase 3.3 framework body pending.")
}
func (r *MetricResource) Read(_ context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.AddError("launchdarkly_metric scaffold", "see Create.")
}
func (r *MetricResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("launchdarkly_metric scaffold", "see Create.")
}
func (r *MetricResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("launchdarkly_metric scaffold", "see Create.")
}
