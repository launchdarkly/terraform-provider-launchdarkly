package launchdarkly

// Frozen prior metric schemas + models used as PriorSchema for state
// upgraders.
//
// v0 (v2.x SDKv2 provider): carried the deprecated `is_active`
// attribute and `randomization_units`.
// v1 (v3 previews up to beta.6): dropped `is_active`, still had
// `randomization_units`.
// v2 (current): renames `randomization_units` to `analysis_units`,
// following the API's randomizationUnits -> analysisUnits rename.
//
// Both upgraders decode prior state into the matching frozen model and
// project to the current MetricResourceModel, discarding IsActive and
// carrying randomization_units over to analysis_units.

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type MetricResourceModelV0 struct {
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

type MetricResourceModelV1 struct {
	ID                        types.String `tfsdk:"id"`
	ProjectKey                types.String `tfsdk:"project_key"`
	Key                       types.String `tfsdk:"key"`
	Name                      types.String `tfsdk:"name"`
	Kind                      types.String `tfsdk:"kind"`
	MaintainerID              types.String `tfsdk:"maintainer_id"`
	Description               types.String `tfsdk:"description"`
	Tags                      types.Set    `tfsdk:"tags"`
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

// metricSchemaAttributesV1 returns the current attribute map with
// `analysis_units` swapped back to the pre-v2 `randomization_units`
// attribute, so the v1->v2 upgrader can decode prior state shapes.
func metricSchemaAttributesV1() map[string]schema.Attribute {
	attrs := metricSchemaAttributes()
	delete(attrs, ANALYSIS_UNITS)
	attrs[RANDOMIZATION_UNITS] = schema.SetAttribute{
		Optional:    true,
		Computed:    true,
		ElementType: types.StringType,
		Description: "A set of one or more context kinds that this metric can measure events from.",
	}
	return attrs
}

// metricSchemaAttributesV0 returns the v1 attribute map plus the
// removed is_active attribute, so the v0->v2 upgrader can decode prior
// state shapes captured under the v2.x SDKv2 provider.
func metricSchemaAttributesV0() map[string]schema.Attribute {
	attrs := metricSchemaAttributesV1()
	attrs[IS_ACTIVE] = schema.BoolAttribute{
		Optional:           true,
		Computed:           true,
		Description:        "Ignored. All metrics are considered active.",
		DeprecationMessage: "No longer in use. This field will be removed in a future major release of the LaunchDarkly provider.",
	}
	return attrs
}
