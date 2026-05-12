package launchdarkly

// Phase 4.3 scaffold for launchdarkly_feature_flag. Heaviest schema in
// the provider — variations, custom_properties (customPropertyHash
// parity, see docs/migration-set-hash-parity.md), client_side_availability
// + deprecated include_in_snippet, defaults block.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &FeatureFlagResource{}

type FeatureFlagResource struct{ client *Client }

type FeatureFlagResourceModel struct {
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

func NewFeatureFlagResource() resource.Resource { return &FeatureFlagResource{} }

func (r *FeatureFlagResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_feature_flag"
}

func (r *FeatureFlagResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly feature flag resource.",
		Attributes: map[string]schema.Attribute{
			"id":           schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			PROJECT_KEY:    schema.StringAttribute{Required: true, Validators: []validator.String{keyValidator()}, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			KEY:            schema.StringAttribute{Required: true, Validators: []validator.String{keyValidator()}, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			NAME:           schema.StringAttribute{Required: true},
			DESCRIPTION:    schema.StringAttribute{Optional: true, Computed: true},
			MAINTAINER_ID:  schema.StringAttribute{Optional: true, Computed: true},
			MAINTAINER_TEAM_KEY: schema.StringAttribute{Optional: true, Computed: true},
			TAGS:           schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType},
			VARIATION_TYPE: schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Validators: []validator.String{oneOfValidator{allowed: []string{"boolean", "string", "number", "json"}}}},
			TEMPORARY: schema.BoolAttribute{
				Optional: true, Computed: true,
				Default: booldefault.StaticBool(false),
			},
			INCLUDE_IN_SNIPPET: schema.BoolAttribute{
				Optional: true, Computed: true,
				DeprecationMessage: "'include_in_snippet' is now deprecated. Please migrate to 'client_side_availability' to maintain future compatability.",
			},
			ARCHIVED:   schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
			DEPRECATED: schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
			VIEW_KEYS:  schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType},
		},
		Blocks: map[string]schema.Block{
			VARIATIONS: schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						NAME:        schema.StringAttribute{Optional: true},
						DESCRIPTION: schema.StringAttribute{Optional: true},
						VALUE:       schema.StringAttribute{Required: true},
					},
				},
			},
			CLIENT_SIDE_AVAILABILITY: schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						USING_ENVIRONMENT_ID: schema.BoolAttribute{Optional: true, Computed: true},
						USING_MOBILE_KEY:     schema.BoolAttribute{Optional: true, Computed: true},
					},
				},
			},
			CUSTOM_PROPERTIES: schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						KEY:   schema.StringAttribute{Required: true},
						NAME:  schema.StringAttribute{Required: true},
						VALUE: schema.ListAttribute{Required: true, ElementType: types.StringType},
					},
				},
			},
			DEFAULTS: schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						ON_VARIATION:  schema.Int64Attribute{Required: true},
						OFF_VARIATION: schema.Int64Attribute{Required: true},
					},
				},
			},
		},
	}
}

func (r *FeatureFlagResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *FeatureFlagResource) Create(_ context.Context, _ resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError("launchdarkly_feature_flag scaffold", "Phase 4.3 framework body pending.")
}
func (r *FeatureFlagResource) Read(_ context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.AddError("launchdarkly_feature_flag scaffold", "see Create.")
}
func (r *FeatureFlagResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("launchdarkly_feature_flag scaffold", "see Create.")
}
func (r *FeatureFlagResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("launchdarkly_feature_flag scaffold", "see Create.")
}
