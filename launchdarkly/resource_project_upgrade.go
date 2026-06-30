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

// environmentModelV0 is the v0 (SDKv2 / pre-REL-14236) environment element
// shape: a positional list element that carried the env key inline. The
// v0->v1 upgrader decodes this and re-keys a map by `Key` (see
// environmentsMapFromV0List).
type environmentModelV0 struct {
	Key                types.String `tfsdk:"key"`
	Name               types.String `tfsdk:"name"`
	Color              types.String `tfsdk:"color"`
	Critical           types.Bool   `tfsdk:"critical"`
	APIKey             types.String `tfsdk:"api_key"`
	MobileKey          types.String `tfsdk:"mobile_key"`
	ClientSideID       types.String `tfsdk:"client_side_id"`
	DefaultTTL         types.Int64  `tfsdk:"default_ttl"`
	SecureMode         types.Bool   `tfsdk:"secure_mode"`
	DefaultTrackEvents types.Bool   `tfsdk:"default_track_events"`
	RequireComments    types.Bool   `tfsdk:"require_comments"`
	ConfirmChanges     types.Bool   `tfsdk:"confirm_changes"`
	Tags               types.Set    `tfsdk:"tags"`
	ApprovalSettings   types.List   `tfsdk:"approval_settings"`
}

func projectSchemaAttributesV0() map[string]schema.Attribute {
	attrs := projectSchemaAttributes()
	attrs[INCLUDE_IN_SNIPPET] = schema.BoolAttribute{
		Optional:           true,
		Computed:           true,
		Description:        "Whether feature flags created under the project should be available to client-side SDKs by default. Please migrate to `default_client_side_availability` to maintain future compatibility.",
		DeprecationMessage: "'include_in_snippet' is now deprecated. Please migrate to 'default_client_side_availability' to maintain future compatibility.",
	}
	// v0 (SDKv2) stored default_client_side_availability as a block, i.e.
	// a single-element list in state. The current (v3) schema models it
	// as a single object. Pin the prior schema to the original list shape
	// so genuine v2.x state still decodes; the upgrader body projects the
	// list to an object via csaObjectFromV0List.
	attrs[DEFAULT_CLIENT_SIDE_AVAILABILITY] = schema.ListNestedAttribute{
		Optional: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				USING_ENVIRONMENT_ID: schema.BoolAttribute{Required: true},
				USING_MOBILE_KEY:     schema.BoolAttribute{Required: true},
			},
		},
	}
	// v0 (SDKv2) stored environments as a positional list whose elements
	// carried the env key inline. The current (v3) schema models it as a
	// map keyed by env key. Pin the prior schema to the original list shape
	// so genuine v2.x state still decodes; the upgrader body re-keys via
	// environmentsMapFromV0List.
	attrs[ENVIRONMENTS] = projectEnvironmentsAttributeV0()
	return attrs
}

// projectEnvironmentsAttributeV0 reproduces the pre-REL-14236 (v0)
// environments list shape — a list of objects each holding its own `key`
// — used only as PriorSchema for the v0->v1 state upgrader.
func projectEnvironmentsAttributeV0() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Optional: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				KEY:                  schema.StringAttribute{Required: true},
				NAME:                 schema.StringAttribute{Required: true},
				COLOR:                schema.StringAttribute{Required: true},
				CRITICAL:             schema.BoolAttribute{Optional: true, Computed: true},
				API_KEY:              schema.StringAttribute{Computed: true, Sensitive: true},
				MOBILE_KEY:           schema.StringAttribute{Computed: true, Sensitive: true},
				CLIENT_SIDE_ID:       schema.StringAttribute{Computed: true, Sensitive: true},
				DEFAULT_TTL:          schema.Int64Attribute{Optional: true, Computed: true},
				SECURE_MODE:          schema.BoolAttribute{Optional: true, Computed: true},
				DEFAULT_TRACK_EVENTS: schema.BoolAttribute{Optional: true, Computed: true},
				REQUIRE_COMMENTS:     schema.BoolAttribute{Optional: true, Computed: true},
				CONFIRM_CHANGES:      schema.BoolAttribute{Optional: true, Computed: true},
				TAGS:                 schema.SetAttribute{Optional: true, ElementType: types.StringType},
				APPROVAL_SETTINGS:    frameworkApprovalSettingsResourceAttribute(),
			},
		},
	}
}
