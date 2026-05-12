package launchdarkly

// resource_destination_framework.go is the Phase 3.1 scaffold. The
// model + schema reflect the SDKv2 surface in
// resource_launchdarkly_destination.go; CRUD is intentionally left as
// a TODO marker so the resource still runs via the SDKv2 path on the
// production mux until the migration body is ready to land.
//
// Promotion checklist (move from scaffold -> done):
//   1. Port Create / Read / Update / Delete from SDKv2.
//   2. Wire destinationConfigFromResourceData / configDiffSuppressFunc
//      into framework's plan-modifier equivalents.
//   3. Register NewDestinationResource on plugin_provider.go's
//      Resources() factory list.
//   4. Remove launchdarkly_destination from provider.go's ResourcesMap.
//   5. Delete the SDKv2 file and the destination_helper.go shim that
//      becomes unreachable.
//   6. Capture state-fixture per the per-PR checklist.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &DestinationResource{}

type DestinationResource struct {
	client *Client
}

type DestinationResourceModel struct {
	ID         types.String `tfsdk:"id"`
	ProjectKey types.String `tfsdk:"project_key"`
	EnvKey     types.String `tfsdk:"env_key"`
	Name       types.String `tfsdk:"name"`
	Kind       types.String `tfsdk:"kind"`
	Config     types.Map    `tfsdk:"config"`
	On         types.Bool   `tfsdk:"on"`
	Tags       types.Set    `tfsdk:"tags"`
}

func NewDestinationResource() resource.Resource {
	return &DestinationResource{}
}

func (r *DestinationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_destination"
}

func (r *DestinationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly Data Export Destination resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:      true,
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			ENV_KEY: schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			NAME: schema.StringAttribute{Required: true},
			KIND: schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					oneOfValidator{allowed: []string{"kinesis", "google-pubsub", "mparticle", "azure-event-hubs", "segment"}},
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			CONFIG: schema.MapAttribute{
				Required:    true,
				ElementType: types.StringType,
			},
			ON: schema.BoolAttribute{Optional: true},
			TAGS: schema.SetAttribute{
				Optional: true, Computed: true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *DestinationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

// CRUD methods are scaffolded as compile-passing no-ops. The resource
// is NOT yet registered in plugin_provider.go::Resources() — until that
// promotion happens, launchdarkly_destination continues to be served by
// the SDKv2 mux side.

func (r *DestinationResource) Create(_ context.Context, _ resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError(
		"launchdarkly_destination scaffold",
		"Phase 3.1 framework resource is staged but not yet wired. Use the SDKv2 mux path until promotion.",
	)
}

func (r *DestinationResource) Read(_ context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.AddError("launchdarkly_destination scaffold", "see Create.")
}

func (r *DestinationResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("launchdarkly_destination scaffold", "see Create.")
}

func (r *DestinationResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("launchdarkly_destination scaffold", "see Create.")
}
