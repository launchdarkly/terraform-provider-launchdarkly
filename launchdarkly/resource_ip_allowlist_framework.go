package launchdarkly

// Phase 3.9 scaffold for launchdarkly_ip_allowlist_config +
// launchdarkly_ip_allowlist_entry. CRUD bodies pending.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource = &IPAllowlistConfigResource{}
	_ resource.Resource = &IPAllowlistEntryResource{}
)

type IPAllowlistConfigResource struct{ client *Client }

type IPAllowlistConfigResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Enabled   types.Bool   `tfsdk:"enabled"`
	EnforceAt types.String `tfsdk:"enforced_at"`
}

func NewIPAllowlistConfigResource() resource.Resource { return &IPAllowlistConfigResource{} }

func (r *IPAllowlistConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ip_allowlist_config"
}

func (r *IPAllowlistConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the account-wide IP allowlist enforcement setting.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			ENABLED:       schema.BoolAttribute{Required: true},
			"enforced_at": schema.StringAttribute{Optional: true, Computed: true},
		},
	}
}

func (r *IPAllowlistConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *IPAllowlistConfigResource) Create(_ context.Context, _ resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError("launchdarkly_ip_allowlist_config scaffold", "Phase 3.9 body pending.")
}
func (r *IPAllowlistConfigResource) Read(_ context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.AddError("launchdarkly_ip_allowlist_config scaffold", "see Create.")
}
func (r *IPAllowlistConfigResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("launchdarkly_ip_allowlist_config scaffold", "see Create.")
}
func (r *IPAllowlistConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("launchdarkly_ip_allowlist_config scaffold", "see Create.")
}

// --- ip_allowlist_entry ---

type IPAllowlistEntryResource struct{ client *Client }

type IPAllowlistEntryResourceModel struct {
	ID          types.String `tfsdk:"id"`
	CIDRBlock   types.String `tfsdk:"cidr_block"`
	Description types.String `tfsdk:"description"`
}

func NewIPAllowlistEntryResource() resource.Resource { return &IPAllowlistEntryResource{} }

func (r *IPAllowlistEntryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ip_allowlist_entry"
}

func (r *IPAllowlistEntryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An entry in the LaunchDarkly IP allowlist.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"cidr_block":  schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			DESCRIPTION:   schema.StringAttribute{Optional: true, Computed: true},
		},
	}
}

func (r *IPAllowlistEntryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *IPAllowlistEntryResource) Create(_ context.Context, _ resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError("launchdarkly_ip_allowlist_entry scaffold", "Phase 3.9 body pending.")
}
func (r *IPAllowlistEntryResource) Read(_ context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.AddError("launchdarkly_ip_allowlist_entry scaffold", "see Create.")
}
func (r *IPAllowlistEntryResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("launchdarkly_ip_allowlist_entry scaffold", "see Create.")
}
func (r *IPAllowlistEntryResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("launchdarkly_ip_allowlist_entry scaffold", "see Create.")
}
