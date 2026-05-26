package launchdarkly

// Frozen pre-v3 project schema + model used as PriorSchema for the
// v0->v1 state upgrader. The v0 shape (v2.x SDKv2 provider) carried
// the deprecated `include_in_snippet` attribute; v3 drops it. The
// upgrader decodes prior state into ProjectResourceModelV0 and
// projects to the current ProjectResourceModel, materializing
// `default_client_side_availability` from `include_in_snippet` when
// DCSA was absent in prior state.

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ProjectResourceModelV0 struct {
	ID                                   types.String `tfsdk:"id"`
	Key                                  types.String `tfsdk:"key"`
	Name                                 types.String `tfsdk:"name"`
	IncludeInSnippet                     types.Bool   `tfsdk:"include_in_snippet"`
	DefaultClientSideAvailability        types.List   `tfsdk:"default_client_side_availability"`
	Tags                                 types.Set    `tfsdk:"tags"`
	Environments                         types.List   `tfsdk:"environments"`
	RequireViewAssociationForNewFlags    types.Bool   `tfsdk:"require_view_association_for_new_flags"`
	RequireViewAssociationForNewSegments types.Bool   `tfsdk:"require_view_association_for_new_segments"`
}

func projectSchemaAttributesV0() map[string]schema.Attribute {
	attrs := projectSchemaAttributes()
	attrs[INCLUDE_IN_SNIPPET] = schema.BoolAttribute{
		Optional:           true,
		Computed:           true,
		Description:        "Whether feature flags created under the project should be available to client-side SDKs by default. Please migrate to `default_client_side_availability` to maintain future compatibility.",
		DeprecationMessage: "'include_in_snippet' is now deprecated. Please migrate to 'default_client_side_availability' to maintain future compatibility.",
	}
	return attrs
}
