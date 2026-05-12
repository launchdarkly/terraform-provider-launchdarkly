package launchdarkly

// Phase 3.8 scaffold for launchdarkly_view_links + launchdarkly_view_filter_links.
// Both are beta-API association resources. CRUD bodies pending.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource = &ViewLinksResource{}
	_ resource.Resource = &ViewFilterLinksResource{}
)

type ViewLinksResource struct{ client *Client }

type ViewLinksResourceModel struct {
	ID         types.String `tfsdk:"id"`
	ProjectKey types.String `tfsdk:"project_key"`
	ViewKey    types.String `tfsdk:"view_key"`
	FlagKeys   types.Set    `tfsdk:"flag_keys"`
	Segments   types.Set    `tfsdk:"linked_segments"`
}

func NewViewLinksResource() resource.Resource { return &ViewLinksResource{} }

func (r *ViewLinksResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_view_links"
}

func (r *ViewLinksResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly view links resource (associates flags/segments to a view).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			PROJECT_KEY: schema.StringAttribute{Required: true, Validators: []validator.String{keyValidator()}, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"view_key":  schema.StringAttribute{Required: true, Validators: []validator.String{keyValidator()}, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"flag_keys": schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType},
		},
		Blocks: map[string]schema.Block{
			"linked_segments": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						SEGMENT_ENVIRONMENT_ID: schema.StringAttribute{Required: true},
						SEGMENT_KEY:            schema.StringAttribute{Required: true},
					},
				},
			},
		},
	}
}

func (r *ViewLinksResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *ViewLinksResource) Create(_ context.Context, _ resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError("launchdarkly_view_links scaffold", "Phase 3.8 body pending.")
}
func (r *ViewLinksResource) Read(_ context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.AddError("launchdarkly_view_links scaffold", "see Create.")
}
func (r *ViewLinksResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("launchdarkly_view_links scaffold", "see Create.")
}
func (r *ViewLinksResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("launchdarkly_view_links scaffold", "see Create.")
}

// --- view_filter_links ---

type ViewFilterLinksResource struct{ client *Client }

type ViewFilterLinksResourceModel struct {
	ID         types.String `tfsdk:"id"`
	ProjectKey types.String `tfsdk:"project_key"`
	ViewKey    types.String `tfsdk:"view_key"`
	FlagFilter types.String `tfsdk:"flag_filter"`
}

func NewViewFilterLinksResource() resource.Resource { return &ViewFilterLinksResource{} }

func (r *ViewFilterLinksResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_view_filter_links"
}

func (r *ViewFilterLinksResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Filter-based view-links resource.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			PROJECT_KEY:   schema.StringAttribute{Required: true, Validators: []validator.String{keyValidator()}, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"view_key":    schema.StringAttribute{Required: true, Validators: []validator.String{keyValidator()}, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"flag_filter": schema.StringAttribute{Required: true},
		},
	}
}

func (r *ViewFilterLinksResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *ViewFilterLinksResource) Create(_ context.Context, _ resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError("launchdarkly_view_filter_links scaffold", "Phase 3.8 body pending.")
}
func (r *ViewFilterLinksResource) Read(_ context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.AddError("launchdarkly_view_filter_links scaffold", "see Create.")
}
func (r *ViewFilterLinksResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("launchdarkly_view_filter_links scaffold", "see Create.")
}
func (r *ViewFilterLinksResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("launchdarkly_view_filter_links scaffold", "see Create.")
}
