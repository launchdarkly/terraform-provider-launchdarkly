package launchdarkly

// Frozen pre-object-syntax flag_trigger schema + model used as
// PriorSchema for the v0->v1 state upgrader. The v0 shape (v2.x SDKv2
// blocks and 3.0.0-beta lists) stored `instructions` as a single-element
// list; the current schema models it as a single object.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type FlagTriggerResourceModelV0 struct {
	ID             types.String `tfsdk:"id"`
	ProjectKey     types.String `tfsdk:"project_key"`
	EnvKey         types.String `tfsdk:"env_key"`
	FlagKey        types.String `tfsdk:"flag_key"`
	IntegrationKey types.String `tfsdk:"integration_key"`
	Instructions   types.List   `tfsdk:"instructions"`
	TriggerURL     types.String `tfsdk:"trigger_url"`
	MaintainerID   types.String `tfsdk:"maintainer_id"`
	Enabled        types.Bool   `tfsdk:"enabled"`
}

// flagTriggerSchemaAttributesV0 pins `instructions` to the original
// single-element list shape so prior state decodes.
func flagTriggerSchemaAttributesV0() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id":            schema.StringAttribute{Computed: true},
		PROJECT_KEY:     schema.StringAttribute{Required: true},
		ENV_KEY:         schema.StringAttribute{Required: true},
		FLAG_KEY:        schema.StringAttribute{Required: true},
		INTEGRATION_KEY: schema.StringAttribute{Required: true},
		TRIGGER_URL:     schema.StringAttribute{Computed: true, Sensitive: true},
		MAINTAINER_ID:   schema.StringAttribute{Computed: true},
		ENABLED:         schema.BoolAttribute{Required: true},
		INSTRUCTIONS: schema.ListNestedAttribute{
			Required: true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					KIND: schema.StringAttribute{Required: true},
				},
			},
		},
	}
}

// flagTriggerInstructionObjectFromV0List projects a v0 single-element
// instructions list into the v3 single-object shape. Returns a null
// object for null/empty input (defensive — instructions is required).
func flagTriggerInstructionObjectFromV0List(ctx context.Context, l types.List) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	if l.IsNull() || l.IsUnknown() || len(l.Elements()) == 0 {
		return types.ObjectNull(flagTriggerInstructionAttrTypes), diags
	}
	type instructionModel struct {
		Kind types.String `tfsdk:"kind"`
	}
	var models []instructionModel
	diags.Append(l.ElementsAs(ctx, &models, false)...)
	if diags.HasError() || len(models) == 0 {
		return types.ObjectNull(flagTriggerInstructionAttrTypes), diags
	}
	obj, d := types.ObjectValue(flagTriggerInstructionAttrTypes, map[string]attr.Value{
		KIND: models[0].Kind,
	})
	diags.Append(d...)
	return obj, diags
}

func (r *FlagTriggerResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	priorSchema := schema.Schema{Attributes: flagTriggerSchemaAttributesV0()}
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &priorSchema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior FlagTriggerResourceModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}
				instructionsObj, d := flagTriggerInstructionObjectFromV0List(ctx, prior.Instructions)
				resp.Diagnostics.Append(d...)
				if resp.Diagnostics.HasError() {
					return
				}
				data := FlagTriggerResourceModel{
					ID:             prior.ID,
					ProjectKey:     prior.ProjectKey,
					EnvKey:         prior.EnvKey,
					FlagKey:        prior.FlagKey,
					IntegrationKey: prior.IntegrationKey,
					Instructions:   instructionsObj,
					TriggerURL:     prior.TriggerURL,
					MaintainerID:   prior.MaintainerID,
					Enabled:        prior.Enabled,
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			},
		},
	}
}
