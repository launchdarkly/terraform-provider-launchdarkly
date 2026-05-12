package launchdarkly

// resource_project_framework.go — Phase 4.1 scaffold.
//
// Project is the highest-risk migration in Phase 4 because of:
//
//   - customizeProjectDiff -> framework ModifyPlan port (IIS/CSA default-
//     injection edge case must be preserved exactly).
//   - Nested environments block (when project is embedded with envs).
//   - include_in_snippet deprecation carry-forward (route through
//     framework_schema_compat.go's stateSetSkipMissingKey / planSetSkipMissingKey
//     for Upjet compat).
//   - default_client_side_availability with the IIS-vs-CSA conflict.
//   - view-association settings via raw HTTP (not in the OpenAPI client).
//
// Promotion checklist:
//   1. Port customizeProjectDiff to ModifyPlan.
//   2. Wire stateSetSkipMissingKey for include_in_snippet writes.
//   3. Capture IIS / CSA / neither / both fixtures via
//      scripts/capture-state-fixtures/capture.sh.
//   4. Soak on the moonshots branch for a calendar week before promotion.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &ProjectResource{}

type ProjectResource struct{ client *Client }

type ProjectResourceModel struct {
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

func NewProjectResource() resource.Resource { return &ProjectResource{} }

func (r *ProjectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *ProjectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly project resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			KEY: schema.StringAttribute{
				Required:      true,
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			NAME: schema.StringAttribute{Required: true},
			INCLUDE_IN_SNIPPET: schema.BoolAttribute{
				Optional: true, Computed: true,
				DeprecationMessage: "'include_in_snippet' is now deprecated. Please migrate to 'default_client_side_availability' to maintain future compatibility.",
			},
			TAGS:                                      schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType},
			REQUIRE_VIEW_ASSOCIATION_FOR_NEW_FLAGS:    schema.BoolAttribute{Optional: true, Computed: true},
			REQUIRE_VIEW_ASSOCIATION_FOR_NEW_SEGMENTS: schema.BoolAttribute{Optional: true, Computed: true},
		},
		Blocks: map[string]schema.Block{
			DEFAULT_CLIENT_SIDE_AVAILABILITY: schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						USING_ENVIRONMENT_ID: schema.BoolAttribute{Required: true},
						USING_MOBILE_KEY:     schema.BoolAttribute{Required: true},
					},
				},
			},
			ENVIRONMENTS: schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						KEY:                  schema.StringAttribute{Required: true},
						NAME:                 schema.StringAttribute{Required: true},
						COLOR:                schema.StringAttribute{Required: true},
						DEFAULT_TTL:          schema.Int64Attribute{Optional: true, Computed: true},
						SECURE_MODE:          schema.BoolAttribute{Optional: true, Computed: true},
						DEFAULT_TRACK_EVENTS: schema.BoolAttribute{Optional: true, Computed: true},
						REQUIRE_COMMENTS:     schema.BoolAttribute{Optional: true, Computed: true},
						CONFIRM_CHANGES:      schema.BoolAttribute{Optional: true, Computed: true},
						CRITICAL:             schema.BoolAttribute{Optional: true, Computed: true},
						TAGS:                 schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType},
						API_KEY:              schema.StringAttribute{Computed: true, Sensitive: true},
						MOBILE_KEY:           schema.StringAttribute{Computed: true, Sensitive: true},
						CLIENT_SIDE_ID:       schema.StringAttribute{Computed: true, Sensitive: true},
					},
				},
			},
		},
	}
}

func (r *ProjectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

// CRUD methods scaffolded. Real port requires customizeProjectDiff ->
// ModifyPlan with full IIS/CSA-injection logic preserved.

func (r *ProjectResource) Create(_ context.Context, _ resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError("launchdarkly_project scaffold", "Phase 4.1 framework body pending. SDKv2 path continues.")
}
func (r *ProjectResource) Read(_ context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.AddError("launchdarkly_project scaffold", "see Create.")
}
func (r *ProjectResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("launchdarkly_project scaffold", "see Create.")
}
func (r *ProjectResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("launchdarkly_project scaffold", "see Create.")
}
