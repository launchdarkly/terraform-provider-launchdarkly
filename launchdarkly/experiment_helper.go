package launchdarkly

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

// Object attribute type maps for the experiment iteration nested structure.
var (
	experimentMetricAttrTypes = map[string]attr.Type{
		KEY:      types.StringType,
		IS_GROUP: types.BoolType,
	}

	experimentParameterAttrTypes = map[string]attr.Type{
		FLAG_KEY:     types.StringType,
		VARIATION_ID: types.StringType,
	}

	experimentTreatmentAttrTypes = map[string]attr.Type{
		NAME:               types.StringType,
		BASELINE:           types.BoolType,
		ALLOCATION_PERCENT: types.StringType,
		PARAMETERS:         types.ListType{ElemType: types.ObjectType{AttrTypes: experimentParameterAttrTypes}},
	}

	experimentFlagAttrTypes = map[string]attr.Type{
		RULE_ID:                        types.StringType,
		FLAG_CONFIG_VERSION:            types.Int64Type,
		NOT_IN_EXPERIMENT_VARIATION_ID: types.StringType,
	}

	experimentIterationAttrTypes = map[string]attr.Type{
		HYPOTHESIS:                types.StringType,
		CAN_RESHUFFLE_TRAFFIC:     types.BoolType,
		RANDOMIZATION_UNIT:        types.StringType,
		ATTRIBUTES:                types.SetType{ElemType: types.StringType},
		PRIMARY_SINGLE_METRIC_KEY: types.StringType,
		PRIMARY_FUNNEL_KEY:        types.StringType,
		METRICS:                   types.ListType{ElemType: types.ObjectType{AttrTypes: experimentMetricAttrTypes}},
		TREATMENTS:                types.ListType{ElemType: types.ObjectType{AttrTypes: experimentTreatmentAttrTypes}},
		FLAGS:                     types.MapType{ElemType: types.ObjectType{AttrTypes: experimentFlagAttrTypes}},
	}
)

type experimentIterationModel struct {
	Hypothesis             types.String `tfsdk:"hypothesis"`
	CanReshuffleTraffic    types.Bool   `tfsdk:"can_reshuffle_traffic"`
	RandomizationUnit      types.String `tfsdk:"randomization_unit"`
	Attributes             types.Set    `tfsdk:"attributes"`
	PrimarySingleMetricKey types.String `tfsdk:"primary_single_metric_key"`
	PrimaryFunnelKey       types.String `tfsdk:"primary_funnel_key"`
	Metrics                types.List   `tfsdk:"metrics"`
	Treatments             types.List   `tfsdk:"treatments"`
	Flags                  types.Map    `tfsdk:"flags"`
}

type experimentMetricModel struct {
	Key     types.String `tfsdk:"key"`
	IsGroup types.Bool   `tfsdk:"is_group"`
}

type experimentTreatmentModel struct {
	Name              types.String `tfsdk:"name"`
	Baseline          types.Bool   `tfsdk:"baseline"`
	AllocationPercent types.String `tfsdk:"allocation_percent"`
	Parameters        types.List   `tfsdk:"parameters"`
}

type experimentParameterModel struct {
	FlagKey     types.String `tfsdk:"flag_key"`
	VariationID types.String `tfsdk:"variation_id"`
}

type experimentFlagModel struct {
	RuleID                     types.String `tfsdk:"rule_id"`
	FlagConfigVersion          types.Int64  `tfsdk:"flag_config_version"`
	NotInExperimentVariationID types.String `tfsdk:"not_in_experiment_variation_id"`
}

// iterationInputFromObject converts the Terraform iteration object into the
// ldapi.IterationInput sent on create and on createIteration.
func iterationInputFromObject(ctx context.Context, iteration types.Object, diags *diag.Diagnostics) ldapi.IterationInput {
	var iter experimentIterationModel
	diags.Append(iteration.As(ctx, &iter, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return ldapi.IterationInput{}
	}

	input := ldapi.IterationInput{
		Hypothesis: iter.Hypothesis.ValueString(),
	}

	if !iter.CanReshuffleTraffic.IsNull() && !iter.CanReshuffleTraffic.IsUnknown() {
		v := iter.CanReshuffleTraffic.ValueBool()
		input.CanReshuffleTraffic = &v
	}
	if v := iter.RandomizationUnit.ValueString(); v != "" {
		input.RandomizationUnit = &v
	}
	if v := iter.PrimarySingleMetricKey.ValueString(); v != "" {
		input.PrimarySingleMetricKey = &v
	}
	if v := iter.PrimaryFunnelKey.ValueString(); v != "" {
		input.PrimaryFunnelKey = &v
	}

	if !iter.Attributes.IsNull() && !iter.Attributes.IsUnknown() {
		attrs, d := stringSliceFromSet(ctx, iter.Attributes)
		diags.Append(d...)
		input.Attributes = attrs
	}

	var metrics []experimentMetricModel
	diags.Append(iter.Metrics.ElementsAs(ctx, &metrics, false)...)
	input.Metrics = make([]ldapi.MetricInput, 0, len(metrics))
	for _, m := range metrics {
		mi := ldapi.MetricInput{Key: m.Key.ValueString()}
		if !m.IsGroup.IsNull() && !m.IsGroup.IsUnknown() {
			g := m.IsGroup.ValueBool()
			mi.IsGroup = &g
		}
		input.Metrics = append(input.Metrics, mi)
	}

	var treatments []experimentTreatmentModel
	diags.Append(iter.Treatments.ElementsAs(ctx, &treatments, false)...)
	input.Treatments = make([]ldapi.TreatmentInput, 0, len(treatments))
	for _, t := range treatments {
		var params []experimentParameterModel
		diags.Append(t.Parameters.ElementsAs(ctx, &params, false)...)
		paramInputs := make([]ldapi.TreatmentParameterInput, 0, len(params))
		for _, p := range params {
			paramInputs = append(paramInputs, ldapi.TreatmentParameterInput{
				FlagKey:     p.FlagKey.ValueString(),
				VariationId: p.VariationID.ValueString(),
			})
		}
		input.Treatments = append(input.Treatments, ldapi.TreatmentInput{
			Name:              t.Name.ValueString(),
			Baseline:          t.Baseline.ValueBool(),
			AllocationPercent: t.AllocationPercent.ValueString(),
			Parameters:        paramInputs,
		})
	}

	flags := map[string]experimentFlagModel{}
	diags.Append(iter.Flags.ElementsAs(ctx, &flags, false)...)
	input.Flags = make(map[string]ldapi.FlagInput, len(flags))
	for key, f := range flags {
		fi := ldapi.FlagInput{
			RuleId:            f.RuleID.ValueString(),
			FlagConfigVersion: int32(f.FlagConfigVersion.ValueInt64()),
		}
		if v := f.NotInExperimentVariationID.ValueString(); v != "" {
			fi.NotInExperimentVariationId = &v
		}
		input.Flags[key] = fi
	}

	return input
}

// experimentID builds the composite Terraform ID for an experiment.
func experimentID(projectKey, environmentKey, key string) string {
	return strings.Join([]string{projectKey, environmentKey, key}, "/")
}

// experimentIDToKeys splits a composite experiment ID into its parts.
func experimentIDToKeys(id string) (projectKey, environmentKey, key string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("expected import ID in the format 'project_key/environment_key/experiment_key', got %q", id)
	}
	return parts[0], parts[1], parts[2], nil
}
