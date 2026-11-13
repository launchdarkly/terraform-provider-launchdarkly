package launchdarkly

// Frozen pre-v3 feature_flag_environment schema + model used as
// PriorSchema for the v0->v1 state upgrader. The v0 shape (v2.x SDKv2
// provider) stored `fallthrough` as a block, i.e. a single-element list
// in state; v3 models it as a single object. The upgrader decodes prior
// state into FeatureFlagEnvironmentResourceModelV0 and projects to the
// current model, converting the fallthrough list to an object.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type FeatureFlagEnvironmentResourceModelV0 struct {
	ID             types.String `tfsdk:"id"`
	FlagID         types.String `tfsdk:"flag_id"`
	EnvKey         types.String `tfsdk:"env_key"`
	On             types.Bool   `tfsdk:"on"`
	Targets        types.Set    `tfsdk:"targets"`
	ContextTargets types.Set    `tfsdk:"context_targets"`
	Rules          types.List   `tfsdk:"rules"`
	Prerequisites  types.List   `tfsdk:"prerequisites"`
	Fallthrough    types.List   `tfsdk:"fallthrough"`
	TrackEvents    types.Bool   `tfsdk:"track_events"`
	OffVariation   types.Int64  `tfsdk:"off_variation"`
}

// featureFlagEnvironmentSchemaAttributesV0 pins `fallthrough` to the
// original block (single-element list) shape so genuine v2.x state
// decodes. All other attributes are unchanged from the current schema.
func featureFlagEnvironmentSchemaAttributesV0() map[string]schema.Attribute {
	attrs := featureFlagEnvironmentSchemaAttributes()
	attrs[FALLTHROUGH] = schema.ListNestedAttribute{
		Required: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				VARIATION:       schema.Int64Attribute{Optional: true, Computed: true},
				BUCKET_BY:       schema.StringAttribute{Optional: true},
				CONTEXT_KIND:    schema.StringAttribute{Optional: true, Computed: true},
				ROLLOUT_WEIGHTS: schema.ListAttribute{Optional: true, ElementType: types.Int64Type},
			},
		},
	}
	return attrs
}

// ffeFallthroughObjectFromV0List projects a v0 (SDKv2) single-element
// fallthrough list into the v3 single-object shape. Returns a null
// object for null/empty input (defensive — fallthrough is required).
func ffeFallthroughObjectFromV0List(ctx context.Context, l types.List) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	if l.IsNull() || l.IsUnknown() || len(l.Elements()) == 0 {
		return types.ObjectNull(ffeFallthroughAttrTypes), diags
	}
	type fallthroughModel struct {
		Variation      types.Int64  `tfsdk:"variation"`
		BucketBy       types.String `tfsdk:"bucket_by"`
		ContextKind    types.String `tfsdk:"context_kind"`
		RolloutWeights types.List   `tfsdk:"rollout_weights"`
	}
	var models []fallthroughModel
	diags.Append(l.ElementsAs(ctx, &models, false)...)
	if diags.HasError() || len(models) == 0 {
		return types.ObjectNull(ffeFallthroughAttrTypes), diags
	}
	m := models[0]
	weights := m.RolloutWeights
	if weights.IsNull() || weights.IsUnknown() {
		weights = types.ListNull(types.Int64Type)
	}
	obj, d := types.ObjectValue(ffeFallthroughAttrTypes, map[string]attr.Value{
		VARIATION:       m.Variation,
		BUCKET_BY:       m.BucketBy,
		CONTEXT_KIND:    m.ContextKind,
		ROLLOUT_WEIGHTS: weights,
	})
	diags.Append(d...)
	return obj, diags
}
