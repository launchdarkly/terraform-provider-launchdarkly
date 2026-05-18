package launchdarkly

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var (
	_ resource.Resource                = &TeamResource{}
	_ resource.ResourceWithImportState = &TeamResource{}
)

type TeamResource struct {
	client *Client
}

type TeamResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Key            types.String `tfsdk:"key"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	MemberIDs      types.Set    `tfsdk:"member_ids"`
	Maintainers    types.Set    `tfsdk:"maintainers"`
	CustomRoleKeys types.Set    `tfsdk:"custom_role_keys"`
	RoleAttributes types.Set    `tfsdk:"role_attributes"`
}

func NewTeamResource() resource.Resource { return &TeamResource{} }

func (r *TeamResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

func (r *TeamResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Provides a LaunchDarkly team resource.

This resource allows you to create and manage a team within your LaunchDarkly organization.

-> **Note:** Teams are available to customers on an Enterprise LaunchDarkly plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			KEY: schema.StringAttribute{
				Required:      true,
				Description:   "The team key. A change in this field will force the destruction of the existing resource and the creation of a new one.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			NAME: schema.StringAttribute{
				Required:    true,
				Description: "A human-friendly name for the team.",
			},
			DESCRIPTION: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The team description.",
			},
			MEMBER_IDS: schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "List of member IDs who belong to the team.",
			},
			MAINTAINERS: schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "List of member IDs for users who maintain the team.",
			},
			CUSTOM_ROLE_KEYS: schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "List of custom role keys the team will access. The referenced custom roles must already exist in LaunchDarkly. If they don't, the provider may behave unexpectedly.",
			},
			ROLE_ATTRIBUTES: frameworkRoleAttributesResourceAttribute(),
		},
	}
}

func (r *TeamResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

// Check for various 400/409 issues at plan time
// * Whether team still has members at deletion time
func (r *TeamResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return
	}
	// Destroy plan: plan is null, state is not.
	// Pre-flight for team stil having members at deletion time
	if req.Plan.Raw.IsNull() && !req.State.Raw.IsNull() {
		var state TeamResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		teamKey := state.Key.ValueString()
		if teamKey == "" {
			return
		}
		members, err := getAllTeamMembers(r.client, teamKey)
		if err != nil {
			// Non-Enterprise tokens 403 here. Degrade to a warning so the
			// destroy still proceeds; the existing apply-time 409 path
			// remains as defence-in-depth.
			resp.Diagnostics.AddWarning(
				fmt.Sprintf("could not check team members for %q during plan", teamKey),
				err.Error()+"\n\nApply may still fail with a 409 conflict if this team still has members.",
			)
			return
		}
		if len(members) > 0 {
			resp.Diagnostics.AddAttributeError(
				path.Root(KEY),
				fmt.Sprintf("team %q still has members and cannot be destroyed", teamKey),
				formatStillAssignedTeamMembersHint(members),
			)
		}
		return
	}
	if req.Plan.Raw.IsNull() {
		return
	}
	if !req.State.Raw.IsNull() {
		return
	}
}

func formatStillAssignedTeamMembersHint(items []ldapi.Member) string {
	var b strings.Builder
	b.WriteString("The following members are still assigned to the team:\n")
	for _, item := range items {
		fmt.Fprintf(&b, "  - member %q\n", item.Email)
	}
	b.WriteString("\nRemove the members from the team (edit its launchdarkly_team_member block) before destroying this team.")
	return b.String()
}

func (r *TeamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TeamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key := plan.Key.ValueString()
	name := plan.Name.ValueString()
	description := plan.Description.ValueString()

	memberIDs, d := stringSliceFromSet(ctx, plan.MemberIDs)
	resp.Diagnostics.Append(d...)
	maintainers, d := stringSliceFromSet(ctx, plan.Maintainers)
	resp.Diagnostics.Append(d...)
	customRoleKeys, d := stringSliceFromSet(ctx, plan.CustomRoleKeys)
	resp.Diagnostics.Append(d...)
	roleAttrs, d := frameworkRoleAttributesFromSet(ctx, plan.RoleAttributes)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	maintainTeam := "maintainTeam"
	grants := make([]ldapi.PermissionGrantInput, 0)
	if len(maintainers) > 0 {
		grants = append(grants, ldapi.PermissionGrantInput{
			ActionSet: &maintainTeam,
			MemberIDs: maintainers,
		})
	}

	body := ldapi.TeamPostInput{
		CustomRoleKeys:   customRoleKeys,
		Description:      &description,
		Key:              key,
		MemberIDs:        memberIDs,
		Name:             name,
		PermissionGrants: grants,
		RoleAttributes:   roleAttrs,
	}

	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.TeamsApi.PostTeam(r.client.ctx).TeamPostInput(body).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error when creating team %q", key), err)
		return
	}

	plan.ID = types.StringValue(key)
	r.readIntoModel(ctx, key, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TeamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TeamResourceModel
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

func (r *TeamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state TeamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamKey := plan.Key.ValueString()
	instructions := make([]map[string]interface{}, 0)
	maintainTeam := "maintainTeam"

	if !plan.Name.Equal(state.Name) {
		instructions = append(instructions, map[string]interface{}{"kind": "updateName", "value": plan.Name.ValueString()})
	}
	if !plan.Description.Equal(state.Description) {
		instructions = append(instructions, map[string]interface{}{"kind": "updateDescription", "value": plan.Description.ValueString()})
	}

	if !plan.MemberIDs.Equal(state.MemberIDs) {
		oldArr, d := stringSliceFromSet(ctx, state.MemberIDs)
		resp.Diagnostics.Append(d...)
		newArr, d := stringSliceFromSet(ctx, plan.MemberIDs)
		resp.Diagnostics.Append(d...)
		remove, add := stringAddRemove(oldArr, newArr)
		if len(remove) > 0 {
			instructions = append(instructions, map[string]interface{}{"kind": "removeMembers", "values": remove})
		}
		if len(add) > 0 {
			instructions = append(instructions, map[string]interface{}{"kind": "addMembers", "values": add})
		}
	}

	if !plan.Maintainers.Equal(state.Maintainers) {
		oldArr, d := stringSliceFromSet(ctx, state.Maintainers)
		resp.Diagnostics.Append(d...)
		newArr, d := stringSliceFromSet(ctx, plan.Maintainers)
		resp.Diagnostics.Append(d...)
		remove, add := stringAddRemove(oldArr, newArr)
		if len(remove) > 0 {
			instructions = append(instructions, map[string]interface{}{"kind": "removePermissionGrants", "actionSet": maintainTeam, "memberIDs": remove})
		}
		if len(add) > 0 {
			instructions = append(instructions, map[string]interface{}{"kind": "addPermissionGrants", "actionSet": maintainTeam, "memberIDs": add})
		}
	}

	if !plan.CustomRoleKeys.Equal(state.CustomRoleKeys) {
		oldArr, d := stringSliceFromSet(ctx, state.CustomRoleKeys)
		resp.Diagnostics.Append(d...)
		newArr, d := stringSliceFromSet(ctx, plan.CustomRoleKeys)
		resp.Diagnostics.Append(d...)
		remove, add := stringAddRemove(oldArr, newArr)
		if len(remove) > 0 {
			instructions = append(instructions, map[string]interface{}{"kind": "removeCustomRoles", "values": remove})
		}
		if len(add) > 0 {
			instructions = append(instructions, map[string]interface{}{"kind": "addCustomRoles", "values": add})
		}
	}

	if !plan.RoleAttributes.Equal(state.RoleAttributes) {
		roleAttrs, d := frameworkRoleAttributesFromSet(ctx, plan.RoleAttributes)
		resp.Diagnostics.Append(d...)
		instructions = append(instructions, map[string]interface{}{
			"kind":  "replaceRoleAttributes",
			"value": roleAttrs,
		})
	}

	if resp.Diagnostics.HasError() {
		return
	}

	if len(instructions) > 0 {
		patch := ldapi.TeamPatchInput{
			Comment:      nil,
			Instructions: instructions,
		}
		err := r.client.withConcurrency(r.client.ctx, func() error {
			_, _, e := r.client.ld.TeamsApi.PatchTeam(r.client.ctx, teamKey).TeamPatchInput(patch).Execute()
			return e
		})
		if err != nil {
			addLdapiError(&resp.Diagnostics, fmt.Sprintf("failed to update team %q", teamKey), err)
			return
		}
	}

	r.readIntoModel(ctx, teamKey, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TeamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TeamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, e := r.client.ld.TeamsApi.DeleteTeam(r.client.ctx, data.ID.ValueString()).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("failed to delete team with key %q", data.ID.ValueString()), err)
	}
}

func (r *TeamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(KEY), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *TeamResource) readIntoModel(
	ctx context.Context,
	teamKey string,
	data *TeamResourceModel,
	diags *diag.Diagnostics,
) {
	var team *ldapi.Team
	var res *http.Response
	var err error
	err = r.client.withConcurrency(r.client.ctx, func() error {
		team, res, err = r.client.ld.TeamsApi.GetTeam(r.client.ctx, teamKey).Expand("roles,projects,maintainers,roleAttributes").Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError(fmt.Sprintf("failed to get team %q", teamKey), handleLdapiErr(err).Error())
		return
	}

	// Paginate team members.
	members, err := getAllTeamMembers(r.client, teamKey)
	if err != nil {
		diags.AddError(fmt.Sprintf("failed to get members for team %q", teamKey), err.Error())
		return
	}

	customRoleKeys, err := getAllTeamCustomRoleKeys(r.client, teamKey)
	if err != nil {
		diags.AddError(fmt.Sprintf("failed to get custom roles for team %q", teamKey), err.Error())
		return
	}
	maintainersList, err := getAllTeamMaintainers(r.client, teamKey)
	if err != nil {
		diags.AddError(fmt.Sprintf("failed to get maintainers for team %q", teamKey), err.Error())
		return
	}

	memberIDs := make([]string, len(members))
	for i, m := range members {
		memberIDs[i] = m.Id
	}
	maintainerIDs := make([]string, len(maintainersList))
	for i, m := range maintainersList {
		maintainerIDs[i] = m.Id
	}

	data.ID = types.StringValue(teamKey)
	if team.Key != nil {
		data.Key = types.StringValue(*team.Key)
	} else {
		data.Key = types.StringValue(teamKey)
	}
	if team.Name != nil {
		data.Name = types.StringValue(*team.Name)
	}
	if team.Description != nil {
		data.Description = types.StringValue(*team.Description)
	} else {
		data.Description = types.StringValue("")
	}

	memberIDsSet, d := setFromStringSlice(ctx, memberIDs)
	diags.Append(d...)
	data.MemberIDs = memberIDsSet

	maintainersSet, d := setFromStringSlice(ctx, maintainerIDs)
	diags.Append(d...)
	data.Maintainers = maintainersSet

	crkSet, d := setFromStringSlice(ctx, customRoleKeys)
	diags.Append(d...)
	data.CustomRoleKeys = crkSet

	roleAttrsVal, d := frameworkRoleAttributesValue(ctx, team.RoleAttributes)
	diags.Append(d...)
	data.RoleAttributes = roleAttrsVal
}

// stringAddRemove returns the elements removed from old → new and added.
func stringAddRemove(old, updated []string) (remove, add []string) {
	oldSet := make(map[string]struct{}, len(old))
	for _, s := range old {
		oldSet[s] = struct{}{}
	}
	newSet := make(map[string]struct{}, len(updated))
	for _, s := range updated {
		newSet[s] = struct{}{}
	}
	for _, s := range old {
		if _, ok := newSet[s]; !ok {
			remove = append(remove, s)
		}
	}
	for _, s := range updated {
		if _, ok := oldSet[s]; !ok {
			add = append(add, s)
		}
	}
	return remove, add
}
