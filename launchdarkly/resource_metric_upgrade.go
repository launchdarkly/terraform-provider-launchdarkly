package launchdarkly

// Frozen pre-v3 metric schema + model used as PriorSchema for the
// v0->v1 state upgrader. The v0 shape (v2.x SDKv2 provider) carried
// the deprecated `is_active` attribute; v3 drops it. The upgrader
// decodes prior state into MetricResourceModelV0 and projects to the
// current MetricResourceModel, discarding IsActive.

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

// metricSchemaAttributesV0 returns the current attribute map plus the
// removed is_active attribute, so the v0->v1 upgrader can decode prior
// state shapes captured under the v2.x SDKv2 provider.
func metricSchemaAttributesV0() map[string]schema.Attribute {
	attrs := metricSchemaAttributes()
	attrs[IS_ACTIVE] = schema.BoolAttribute{
		Optional:           true,
		Computed:           true,
		Description:        "Ignored. All metrics are considered active.",
		DeprecationMessage: "No longer in use. This field will be removed in a future major release of the LaunchDarkly provider.",
	}
	return attrs
}
