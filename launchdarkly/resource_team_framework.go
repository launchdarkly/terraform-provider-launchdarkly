package launchdarkly

// Phase 3.7 scaffold for launchdarkly_team. CRUD body pending; framework
// schema + model captured. Reuses frameworkRoleAttributesResourceBlock.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &TeamResource{}

type TeamResource struct{ client *Client }

type TeamResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Key                 types.String `tfsdk:"key"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	Maintainers         types.Set    `tfsdk:"maintainers"`
	Members             types.Set    `tfsdk:"member_ids"`
	ProjectKeys         types.Set    `tfsdk:"project_keys"`
	CustomRoleKeys      types.Set    `tfsdk:"custom_role_keys"`
	RoleAttributes      types.Set    `tfsdk:"role_attributes"`
	PermissionGrants    types.Set    `tfsdk:"permission_grants"`
	NotifyMemberIDs     types.Set    `tfsdk:"notify_member_ids"`
}

func NewTeamResource() resource.Resource { return &TeamResource{} }

func (r *TeamResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

func (r *TeamResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly team resource (Enterprise).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			KEY: schema.StringAttribute{
				Required:      true,
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			NAME:             schema.StringAttribute{Required: true},
			DESCRIPTION:      schema.StringAttribute{Optional: true, Computed: true},
			CUSTOM_ROLE_KEYS: schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType},
			"member_ids":     schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType},
			"permission_grants": schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType},
			"notify_member_ids": schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType},
			"project_keys":   schema.SetAttribute{Computed: true, ElementType: types.StringType},
		},
		Blocks: map[string]schema.Block{
			MAINTAINERS: schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						EMAIL:      schema.StringAttribute{Required: true},
						ID:         schema.StringAttribute{Optional: true, Computed: true},
						FIRST_NAME: schema.StringAttribute{Computed: true},
						LAST_NAME:  schema.StringAttribute{Computed: true},
						ROLE:       schema.StringAttribute{Computed: true},
					},
				},
			},
			ROLE_ATTRIBUTES: frameworkRoleAttributesResourceBlock(),
		},
	}
}

func (r *TeamResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *TeamResource) Create(_ context.Context, _ resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError("launchdarkly_team scaffold", "Phase 3.7 framework body pending.")
}
func (r *TeamResource) Read(_ context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.AddError("launchdarkly_team scaffold", "see Create.")
}
func (r *TeamResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("launchdarkly_team scaffold", "see Create.")
}
func (r *TeamResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("launchdarkly_team scaffold", "see Create.")
}
