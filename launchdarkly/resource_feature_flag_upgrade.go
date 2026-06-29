package launchdarkly

// Frozen pre-v3 feature_flag schema + model used as PriorSchema for
// the v0->v1 state upgrader. The v0 shape (v2.x SDKv2 provider)
// carried the deprecated `include_in_snippet` attribute; v3 drops it.
// The upgrader decodes prior state into FeatureFlagResourceModelV0
// and projects to the current FeatureFlagResourceModel, materializing
// `client_side_availability` from `include_in_snippet` when CSA was
// absent in prior state.

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type FeatureFlagResourceModelV0 struct {
	ID                     types.String `tfsdk:"id"`
	ProjectKey             types.String `tfsdk:"project_key"`
	Key                    types.String `tfsdk:"key"`
	Name                   types.String `tfsdk:"name"`
	Description            types.String `tfsdk:"description"`
	MaintainerID           types.String `tfsdk:"maintainer_id"`
	MaintainerTeamKey      types.String `tfsdk:"maintainer_team_key"`
	Tags                   types.Set    `tfsdk:"tags"`
	VariationType          types.String `tfsdk:"variation_type"`
	Variations             types.List   `tfsdk:"variations"`
	Temporary              types.Bool   `tfsdk:"temporary"`
	IncludeInSnippet       types.Bool   `tfsdk:"include_in_snippet"`
	ClientSideAvailability types.List   `tfsdk:"client_side_availability"`
	CustomProperties       types.Set    `tfsdk:"custom_properties"`
	Defaults               types.List   `tfsdk:"defaults"`
	Archived               types.Bool   `tfsdk:"archived"`
	Deprecated             types.Bool   `tfsdk:"deprecated"`
	ViewKeys               types.Set    `tfsdk:"view_keys"`
}

// featureFlagSchemaAttributesV0 returns the current attribute map
// plus the removed include_in_snippet attribute, so the v0->v1
// upgrader can decode prior state shapes captured under the v2.x
// SDKv2 provider.
func featureFlagSchemaAttributesV0() map[string]schema.Attribute {
	attrs := featureFlagSchemaAttributes()
	attrs[INCLUDE_IN_SNIPPET] = schema.BoolAttribute{
		Optional:           true,
		Computed:           true,
		Description:        "Specifies whether this flag should be made available to the client-side JavaScript SDK using the client-side Id. This value gets its default from your project configuration if not set. `include_in_snippet` is now deprecated. Please migrate to `client_side_availability.using_environment_id` to maintain future compatibility.",
		DeprecationMessage: "'include_in_snippet' is now deprecated. Please migrate to 'client_side_availability' to maintain future compatability.",
	}
	// v0 (SDKv2) stored client_side_availability and defaults as blocks,
	// i.e. single-element lists in state. The current (v3) schema models
	// them as single objects. Pin the prior schema to the original list
	// shape so genuine v2.x state still decodes; the upgrader body
	// projects the lists to objects via csaObjectFromV0List /
	// defaultsObjectFromV0List.
	attrs[CLIENT_SIDE_AVAILABILITY] = schema.ListNestedAttribute{
		Optional: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				USING_ENVIRONMENT_ID: schema.BoolAttribute{Optional: true, Computed: true},
				USING_MOBILE_KEY:     schema.BoolAttribute{Optional: true, Computed: true},
			},
		},
	}
	attrs[DEFAULTS] = schema.ListNestedAttribute{
		Optional: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				ON_VARIATION:  schema.Int64Attribute{Required: true},
				OFF_VARIATION: schema.Int64Attribute{Required: true},
			},
		},
	}
	return attrs
}
