package launchdarkly

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var _ datasource.DataSource = &MetricDataSource{}

type MetricDataSource struct {
	client *Client
}

type MetricDataSourceModel struct {
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

var metricURLAttrTypes = map[string]attr.Type{
	KIND:      types.StringType,
	URL:       types.StringType,
	SUBSTRING: types.StringType,
	PATTERN:   types.StringType,
}

func NewMetricDataSource() datasource.DataSource {
	return &MetricDataSource{}
}

func (d *MetricDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_metric"
}

func (d *MetricDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly metric data source.\n\nThis data source allows you to retrieve metric information from your LaunchDarkly organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID in the format `project_key/key`.",
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The metric's project key.",
			},
			KEY: schema.StringAttribute{
				Required:    true,
				Description: "The unique key that references the metric.",
			},
			NAME: schema.StringAttribute{
				Computed:    true,
				Description: "The human-friendly name for the metric.",
			},
			KIND: schema.StringAttribute{
				Computed:    true,
				Description: "The metric type.",
			},
			MAINTAINER_ID: schema.StringAttribute{
				Computed:    true,
				Description: "The LaunchDarkly member ID of the maintainer.",
			},
			DESCRIPTION: schema.StringAttribute{
				Computed:    true,
				Description: "The description of the metric's purpose.",
			},
			TAGS: schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Tags associated with the metric.",
			},
			IS_ACTIVE: schema.BoolAttribute{
				Computed:           true,
				Description:        "Ignored. All metrics are considered active.",
				DeprecationMessage: "No longer in use. This field will be removed in a future major release of the LaunchDarkly provider.",
			},
			IS_NUMERIC: schema.BoolAttribute{
				Computed:    true,
				Description: "Whether a `custom` metric is a numeric metric or not.",
			},
			UNIT: schema.StringAttribute{
				Computed:    true,
				Description: "The unit for numeric `custom` metrics.",
			},
			SELECTOR: schema.StringAttribute{
				Computed:    true,
				Description: "The CSS selector for your metric (if click metric).",
			},
			EVENT_KEY: schema.StringAttribute{
				Computed:    true,
				Description: "The event key for your metric (if custom metric).",
			},
			SUCCESS_CRITERIA: schema.StringAttribute{
				Computed:    true,
				Description: "The success criteria for your metric (if numeric metric).",
			},
			RANDOMIZATION_UNITS: schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "A set of one or more context kinds that this metric can measure events from.",
			},
			INCLUDE_UNITS_WITHOUT_EVENTS: schema.BoolAttribute{
				Computed:    true,
				Description: "Include units that did not send any events and set their value to 0.",
			},
			UNIT_AGGREGATION_TYPE: schema.StringAttribute{
				Computed:    true,
				Description: "The method by which multiple unit event values are aggregated.",
			},
			ANALYSIS_TYPE: schema.StringAttribute{
				Computed:    true,
				Description: "The method for analyzing metric events.",
			},
			PERCENTILE_VALUE: schema.Int64Attribute{
				Computed:    true,
				Description: "The percentile for the analysis method.",
			},
			VERSION: schema.Int64Attribute{
				Computed:    true,
				Description: "Version of the metric.",
			},
		},
		Blocks: map[string]schema.Block{
			URLS: schema.ListNestedBlock{
				Description: "Nested `url` blocks describing URLs associated with the metric.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						KIND:      schema.StringAttribute{Computed: true, Description: "The URL type."},
						URL:       schema.StringAttribute{Computed: true, Description: "The exact or canonical URL."},
						SUBSTRING: schema.StringAttribute{Computed: true, Description: "The URL substring to match by."},
						PATTERN:   schema.StringAttribute{Computed: true, Description: "The regex pattern to match by."},
					},
				},
			},
		},
	}
}

func (d *MetricDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *MetricDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data MetricDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	key := data.Key.ValueString()

	var metric *ldapi.MetricRep
	var res *http.Response
	var err error
	err = d.client.withConcurrency(d.client.ctx, func() error {
		metric, res, err = d.client.ld.MetricsApi.GetMetric(d.client.ctx, projectKey, key).Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			resp.Diagnostics.AddError("Metric not found", fmt.Sprintf("Metric %q in project %q not found.", key, projectKey))
			return
		}
		addLdapiError(&resp.Diagnostics, "Failed to get metric", err)
		return
	}

	data.ID = types.StringValue(projectKey + "/" + key)
	data.Key = types.StringValue(metric.Key)
	data.Name = types.StringValue(metric.Name)
	data.Kind = types.StringValue(metric.Kind)
	if metric.MaintainerId != nil {
		data.MaintainerID = types.StringValue(*metric.MaintainerId)
	} else {
		data.MaintainerID = types.StringValue("")
	}
	if metric.Description != nil {
		data.Description = types.StringValue(*metric.Description)
	} else {
		data.Description = types.StringValue("")
	}
	if metric.IsActive != nil {
		data.IsActive = types.BoolValue(*metric.IsActive)
	} else {
		data.IsActive = types.BoolValue(true)
	}
	if metric.IsNumeric != nil {
		data.IsNumeric = types.BoolValue(*metric.IsNumeric)
	} else {
		data.IsNumeric = types.BoolValue(false)
	}
	if metric.Unit != nil {
		data.Unit = types.StringValue(*metric.Unit)
	} else {
		data.Unit = types.StringValue("")
	}
	if metric.Selector != nil {
		data.Selector = types.StringValue(*metric.Selector)
	} else {
		data.Selector = types.StringValue("")
	}
	if metric.EventKey != nil {
		data.EventKey = types.StringValue(*metric.EventKey)
	} else {
		data.EventKey = types.StringValue("")
	}
	if metric.SuccessCriteria != nil {
		data.SuccessCriteria = types.StringValue(*metric.SuccessCriteria)
	} else {
		data.SuccessCriteria = types.StringValue("")
	}
	if metric.Version != nil {
		data.Version = types.Int64Value(int64(*metric.Version))
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
		data.PercentileValue = types.Int64Null()
	}
	if metric.EventDefault != nil && metric.EventDefault.Disabled != nil {
		data.IncludeUnitsWithoutEvents = types.BoolValue(!*metric.EventDefault.Disabled)
	} else {
		data.IncludeUnitsWithoutEvents = types.BoolValue(false)
	}

	tagsSet, diags := setFromStringSlice(ctx, metric.Tags)
	resp.Diagnostics.Append(diags...)
	data.Tags = tagsSet

	randomization, diags := setFromStringSlice(ctx, metric.RandomizationUnits)
	resp.Diagnostics.Append(diags...)
	data.RandomizationUnits = randomization

	objectType := types.ObjectType{AttrTypes: metricURLAttrTypes}
	elements := make([]attr.Value, 0, len(metric.Urls))
	for _, u := range metric.Urls {
		kindVal := types.StringNull()
		urlVal := types.StringNull()
		subVal := types.StringNull()
		patVal := types.StringNull()
		if v, ok := u["kind"].(string); ok {
			kindVal = types.StringValue(v)
		}
		if v, ok := u["url"].(string); ok {
			urlVal = types.StringValue(v)
		}
		if v, ok := u["substring"].(string); ok {
			subVal = types.StringValue(v)
		}
		if v, ok := u["pattern"].(string); ok {
			patVal = types.StringValue(v)
		}
		obj, d := types.ObjectValue(metricURLAttrTypes, map[string]attr.Value{
			KIND: kindVal, URL: urlVal, SUBSTRING: subVal, PATTERN: patVal,
		})
		resp.Diagnostics.Append(d...)
		elements = append(elements, obj)
	}
	list, diags := types.ListValue(objectType, elements)
	resp.Diagnostics.Append(diags...)
	data.URLs = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
