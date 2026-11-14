package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const ipAllowlistConfigID = "ip-allowlist-config"

// -----------------------------------------------------------------------------
// launchdarkly_ip_allowlist_config
// -----------------------------------------------------------------------------

var (
	_ resource.Resource                = &IPAllowlistConfigResource{}
	_ resource.ResourceWithImportState = &IPAllowlistConfigResource{}
)

type IPAllowlistConfigResource struct {
	client *Client
}

type IPAllowlistConfigResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	SessionAllowlistEnabled types.Bool   `tfsdk:"session_allowlist_enabled"`
	ScopedAllowlistEnabled  types.Bool   `tfsdk:"scoped_allowlist_enabled"`
}

func NewIPAllowlistConfigResource() resource.Resource { return &IPAllowlistConfigResource{} }

func (r *IPAllowlistConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ip_allowlist_config"
}

func (r *IPAllowlistConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Provides a LaunchDarkly IP allowlist configuration resource.

-> **Note:** IP allowlists are available to customers on an Enterprise LaunchDarkly plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).

~> **Beta:** This resource uses a beta API. Beta resources may change or be removed in future versions.

This resource allows you to manage the IP allowlist configuration for your LaunchDarkly account. There is only one configuration per account, so you should define only a single instance of this resource.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			SESSION_ALLOWLIST_ENABLED: schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the session IP allowlist is enabled.",
			},
			SCOPED_ALLOWLIST_ENABLED: schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the scoped (API token) IP allowlist is enabled.",
			},
		},
	}
}

func (r *IPAllowlistConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *IPAllowlistConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan IPAllowlistConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	session := plan.SessionAllowlistEnabled.ValueBool()
	scoped := plan.ScopedAllowlistEnabled.ValueBool()
	if _, err := patchIpAllowlistConfig(r.client, &session, &scoped); err != nil {
		resp.Diagnostics.AddError("Failed to create IP allowlist config", err.Error())
		return
	}

	plan.ID = types.StringValue(ipAllowlistConfigID)
	r.readIntoModel(&plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IPAllowlistConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data IPAllowlistConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readIntoModel(&data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IPAllowlistConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan IPAllowlistConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	session := plan.SessionAllowlistEnabled.ValueBool()
	scoped := plan.ScopedAllowlistEnabled.ValueBool()
	if _, err := patchIpAllowlistConfig(r.client, &session, &scoped); err != nil {
		resp.Diagnostics.AddError("Failed to update IP allowlist config", err.Error())
		return
	}

	r.readIntoModel(&plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete resets both allowlist flags to false. The resource is a
// singleton; destroying the TF resource reverts the server-side config
// to defaults rather than deleting anything.
func (r *IPAllowlistConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	falseVal := false
	if _, err := patchIpAllowlistConfig(r.client, &falseVal, &falseVal); err != nil {
		resp.Diagnostics.AddError("Failed to reset IP allowlist config", err.Error())
	}
}

func (r *IPAllowlistConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *IPAllowlistConfigResource) readIntoModel(data *IPAllowlistConfigResourceModel, diags *diag.Diagnostics) {
	allowlist, err := getIpAllowlist(r.client)
	if err != nil {
		diags.AddError("Failed to read IP allowlist config", err.Error())
		return
	}
	data.ID = types.StringValue(ipAllowlistConfigID)
	data.SessionAllowlistEnabled = types.BoolValue(allowlist.SessionAllowlistEnabled)
	data.ScopedAllowlistEnabled = types.BoolValue(allowlist.ApiTokenAllowlistEnabled)
}

// -----------------------------------------------------------------------------
// launchdarkly_ip_allowlist_entry
// -----------------------------------------------------------------------------

var (
	_ resource.Resource                = &IPAllowlistEntryResource{}
	_ resource.ResourceWithImportState = &IPAllowlistEntryResource{}
)

type IPAllowlistEntryResource struct {
	client *Client
}

type IPAllowlistEntryResourceModel struct {
	ID          types.String `tfsdk:"id"`
	IPAddress   types.String `tfsdk:"ip_address"`
	Description types.String `tfsdk:"description"`
}

func NewIPAllowlistEntryResource() resource.Resource { return &IPAllowlistEntryResource{} }

func (r *IPAllowlistEntryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ip_allowlist_entry"
}

func (r *IPAllowlistEntryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Provides a LaunchDarkly IP allowlist entry resource.

-> **Note:** IP allowlists are available to customers on an Enterprise LaunchDarkly plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).

~> **Beta:** This resource uses a beta API. Beta resources may change or be removed in future versions.

This resource allows you to create and manage IP allowlist entries within your LaunchDarkly account.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			IP_ADDRESS: schema.StringAttribute{
				Required:      true,
				Description:   "The IP address or CIDR block for the allowlist entry. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			DESCRIPTION: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "A human-readable description of the IP allowlist entry.",
			},
		},
	}
}

func (r *IPAllowlistEntryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *IPAllowlistEntryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan IPAllowlistEntryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var description *string
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() && plan.Description.ValueString() != "" {
		s := plan.Description.ValueString()
		description = &s
	}
	entry, err := createIpAllowlistEntry(r.client, plan.IPAddress.ValueString(), description)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create IP allowlist entry", err.Error())
		return
	}

	plan.ID = types.StringValue(entry.Id)
	r.readIntoModel(plan.ID.ValueString(), &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IPAllowlistEntryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data IPAllowlistEntryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readIntoModel(data.ID.ValueString(), &data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IPAllowlistEntryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state IPAllowlistEntryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.Description.Equal(state.Description) {
		desc := plan.Description.ValueString()
		if _, err := patchIpAllowlistEntry(r.client, state.ID.ValueString(), desc); err != nil {
			resp.Diagnostics.AddError("Failed to update IP allowlist entry", err.Error())
			return
		}
	}

	plan.ID = state.ID
	r.readIntoModel(plan.ID.ValueString(), &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IPAllowlistEntryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data IPAllowlistEntryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := deleteIpAllowlistEntry(r.client, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete IP allowlist entry", err.Error())
	}
}

func (r *IPAllowlistEntryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *IPAllowlistEntryResource) readIntoModel(id string, data *IPAllowlistEntryResourceModel, diags *diag.Diagnostics) {
	allowlist, err := getIpAllowlist(r.client)
	if err != nil {
		diags.AddError("Failed to read IP allowlist", err.Error())
		return
	}
	entry := findIpAllowlistEntryByID(allowlist.Entries, id)
	if entry == nil {
		data.ID = types.StringNull()
		return
	}
	data.ID = types.StringValue(entry.Id)
	data.IPAddress = types.StringValue(entry.IpAddress)
	data.Description = stringValueFromPointer(entry.Description)
}
