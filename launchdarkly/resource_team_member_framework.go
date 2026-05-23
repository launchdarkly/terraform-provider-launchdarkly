package launchdarkly

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var (
	_ resource.Resource                     = &TeamMemberResource{}
	_ resource.ResourceWithImportState      = &TeamMemberResource{}
	_ resource.ResourceWithConfigValidators = &TeamMemberResource{}
	_ resource.ResourceWithUpgradeState     = &TeamMemberResource{}
)

type TeamMemberResource struct {
	client *Client
}

type TeamMemberResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Email          types.String `tfsdk:"email"`
	FirstName      types.String `tfsdk:"first_name"`
	LastName       types.String `tfsdk:"last_name"`
	Role           types.String `tfsdk:"role"`
	CustomRoles    types.Set    `tfsdk:"custom_roles"`
	RoleAttributes types.Set    `tfsdk:"role_attributes"`
}

func NewTeamMemberResource() resource.Resource {
	return &TeamMemberResource{}
}

func (r *TeamMemberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_member"
}

func (r *TeamMemberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: 1,
		Description: `Provides a LaunchDarkly team member resource.

This resource allows you to create and manage team members within your LaunchDarkly organization.

-> **Note:** You can only manage team members with "admin" level personal access tokens. To learn more, read [Managing Teams](https://docs.launchdarkly.com/home/teams/managing).`,
		Attributes: teamMemberSchemaAttributes(),
	}
}

func teamMemberSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:      true,
			Description:   "The 24 character alphanumeric ID of the team member.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		EMAIL: schema.StringAttribute{
			Required:    true,
			Description: addForceNewDescription("The unique email address associated with the team member.", true),
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		FIRST_NAME: schema.StringAttribute{
			Optional:    true,
			Description: "The team member's given name. Once created, this cannot be updated except by the team member.",
		},
		LAST_NAME: schema.StringAttribute{
			Optional:    true,
			Description: "TThe team member's family name. Once created, this cannot be updated except by the team member.",
		},
		ROLE: schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "The role associated with team member. Supported roles are `reader`, `writer`, `no_access`, or `admin`. If you don't specify a role, `reader` is assigned by default.",
			Validators: []validator.String{
				oneOfValidator{allowed: []string{"reader", "writer", "admin", "no_access"}},
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		CUSTOM_ROLES: schema.SetAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: "The list of custom roles keys associated with the team member. Custom roles are only available to customers on an Enterprise plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).\n\n-> **Note:** each `launchdarkly_team_member` must have either a `role` or `custom_roles` argument.",
		},
		ROLE_ATTRIBUTES: frameworkRoleAttributesResourceAttribute(),
	}
}

func (r *TeamMemberResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	priorSchema := schema.Schema{Attributes: teamMemberSchemaAttributes()}
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &priorSchema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var data TeamMemberResourceModel
				resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
				if resp.Diagnostics.HasError() {
					return
				}
				data.CustomRoles = nullIfEmptySet(ctx, data.CustomRoles)
				data.RoleAttributes = nullIfEmptySet(ctx, data.RoleAttributes)
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			},
		},
	}
}

func (r *TeamMemberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *TeamMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TeamMemberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	email := plan.Email.ValueString()
	firstName := plan.FirstName.ValueString()
	lastName := plan.LastName.ValueString()
	role := plan.Role.ValueString()

	customRoles, diags := stringSliceFromSet(ctx, plan.CustomRoles)
	resp.Diagnostics.Append(diags...)

	roleAttrs, diags := frameworkRoleAttributesFromSet(ctx, plan.RoleAttributes)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	memberForm := ldapi.NewMemberForm{
		Email:          email,
		FirstName:      &firstName,
		LastName:       &lastName,
		Role:           &role,
		CustomRoles:    customRoles,
		RoleAttributes: roleAttrs,
	}

	var members *ldapi.Members
	err := r.client.withConcurrency(r.client.ctx, func() error {
		var e error
		members, _, e = r.client.ld.AccountMembersApi.PostMembers(r.client.ctx).NewMemberForm([]ldapi.NewMemberForm{memberForm}).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to create team member", err)
		return
	}
	if len(members.Items) == 0 {
		resp.Diagnostics.AddError("No member returned", "Create returned an empty Items list.")
		return
	}
	plan.ID = types.StringValue(members.Items[0].Id)

	r.readIntoModel(ctx, plan.ID.ValueString(), &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TeamMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TeamMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readIntoModel(ctx, data.ID.ValueString(), &data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TeamMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state TeamMemberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	memberID := plan.ID.ValueString()
	role := plan.Role.ValueString()
	customRoleKeys, diags := stringSliceFromSet(ctx, plan.CustomRoles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	customRoleIds, err := customRoleKeysToIDs(r.client, customRoleKeys)
	if err != nil {
		resp.Diagnostics.AddError("Failed to look up custom role IDs", err.Error())
		return
	}

	patch := []ldapi.PatchOperation{
		patchReplace("/role", &role),
		patchReplace("/customRoles", &customRoleIds),
	}
	patch = append(patch, frameworkRoleAttributePatches(ctx, plan.RoleAttributes, state.RoleAttributes)...)

	err = r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.AccountMembersApi.PatchMember(r.client.ctx, memberID).PatchOperation(patch).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to update team member", err)
		return
	}

	r.readIntoModel(ctx, memberID, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TeamMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TeamMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, e := r.client.ld.AccountMembersApi.DeleteMember(r.client.ctx, data.ID.ValueString()).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to delete team member", err)
	}
}

func (r *TeamMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *TeamMemberResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{teamMemberRoleOrCustomRolesValidator{}}
}

// teamMemberRoleOrCustomRolesValidator requires at least one of role
// or custom_roles to be set.
type teamMemberRoleOrCustomRolesValidator struct{}

func (teamMemberRoleOrCustomRolesValidator) Description(context.Context) string {
	return "at least one of role or custom_roles must be set"
}

func (teamMemberRoleOrCustomRolesValidator) MarkdownDescription(ctx context.Context) string {
	return teamMemberRoleOrCustomRolesValidator{}.Description(ctx)
}

func (teamMemberRoleOrCustomRolesValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data TeamMemberResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	roleSet := !data.Role.IsNull() && !data.Role.IsUnknown() && data.Role.ValueString() != ""
	customRolesSet := !data.CustomRoles.IsNull() && !data.CustomRoles.IsUnknown() && len(data.CustomRoles.Elements()) > 0
	if !roleSet && !customRolesSet {
		resp.Diagnostics.AddError(
			"Missing required attribute",
			"At least one of role or custom_roles must be set.",
		)
	}
}

func (r *TeamMemberResource) readIntoModel(
	ctx context.Context,
	memberID string,
	data *TeamMemberResourceModel,
	diags *diag.Diagnostics,
) {
	var member *ldapi.Member
	var res *http.Response
	var err error
	err = r.client.withConcurrency(r.client.ctx, func() error {
		member, res, err = r.client.ld.AccountMembersApi.GetMember(r.client.ctx, memberID).Expand("roleAttributes").Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError("Failed to get team member", handleLdapiErr(err).Error())
		return
	}
	data.ID = types.StringValue(member.Id)
	data.Email = types.StringValue(member.Email)
	// Optional-only attrs: null-when-empty for plan-apply consistency.
	data.FirstName = stringValueOrNullFromPointer(member.FirstName)
	data.LastName = stringValueOrNullFromPointer(member.LastName)
	data.Role = types.StringValue(member.Role)

	// API returns custom-role IDs; convert to keys for state.
	customRoleKeys, err := customRoleIDsToKeys(r.client, member.CustomRoles)
	if err != nil {
		diags.AddError("Failed to resolve custom role keys", err.Error())
		return
	}
	// Optional-only Set attr with plan-aware null-vs-empty handling:
	// preserves the user's distinction between `custom_roles = []`
	// (plan empty Set, apply must echo empty Set) and omitted
	// (plan null, apply must echo null). See helper godoc.
	rolesSet, d := setFromStringSlicePreservingPlan(ctx, customRoleKeys, data.CustomRoles)
	diags.Append(d...)
	data.CustomRoles = rolesSet

	attrs, d := frameworkRoleAttributesValue(ctx, member.RoleAttributes)
	diags.Append(d...)
	data.RoleAttributes = attrs
}
