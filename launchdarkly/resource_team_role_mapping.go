package launchdarkly

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v23"
)

var (
	_ resource.Resource = &TeamRoleMappingResource{}
)

type TeamRoleMappingResource struct {
	client *Client
}

type TeamRoleMappingResourceModel struct {
	TeamKey        types.String `tfsdk:"team_key"`
	CustomRoleKeys types.Set    `tfsdk:"custom_role_keys"`
	RoleAttributes types.Map    `tfsdk:"role_attributes"`
	ID             types.String `tfsdk:"id"`
}

func (r *TeamRoleMappingResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_role_mapping"
}

func (r *TeamRoleMappingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{

			"team_key": schema.StringAttribute{
				Description: "The LaunchDarkly team key.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"custom_role_keys": schema.SetAttribute{
				ElementType: types.StringType,
				Description: "A set of custom role keys to assign to the team.",
				Required:    true,
			},
			ROLE_ATTRIBUTES: schema.MapAttribute{
				ElementType: types.ListType{ElemType: types.StringType},
				Optional:    true,
				Description: "Map of role-attribute keys to lists of resource keys. Applied to the team as a whole — every custom role granted to this team gets these scopes (see https://launchdarkly.com/docs/home/account/roles/role-scope). Conflicts with `role_attributes` on `launchdarkly_team`; if you manage the team via `launchdarkly_team`, set `role_attributes` there instead, or add `lifecycle { ignore_changes = [role_attributes] }` on the `launchdarkly_team` to avoid plan churn.",
			},
			// Framework resources require an explicit id attribute; it
			// is conventionally Computed and holds the resource identifier.
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID for this resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Description: "Manages the mapping of LaunchDarkly custom roles to teams.",
	}
}

func (r *TeamRoleMappingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *TeamRoleMappingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *TeamRoleMappingResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	teamKey := data.TeamKey.ValueString()
	var team *ldapi.Team
	var res *http.Response
	var err error
	err = r.client.withConcurrency(r.client.ctx, func() error {
		team, res, err = r.client.ld404Retry.TeamsApi.GetTeam(r.client.ctx, teamKey).Expand("roles,roleAttributes").Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			resp.Diagnostics.AddError("Team not found", fmt.Sprintf("Unable to create the team/role mapping because the team %q does not exist.", teamKey))
			return
		}

		resp.Diagnostics.AddError("Unable to get team", fmt.Sprintf("Received an error when fetching the team %q: %s", teamKey, handleLdapiErr((err))))
		return
	}

	data.TeamKey = types.StringValue(*team.Key)
	data.ID = types.StringValue(*team.Key)

	// Check if team already has custom roles assigned using paginated API
	existingRoleKeys, err := getAllTeamCustomRoleKeysWithRetry(r.client, teamKey)
	if err != nil {
		resp.Diagnostics.AddError("Unable to get team roles", fmt.Sprintf("Received an error when fetching roles for team %q: %s", teamKey, err))
		return
	}
	if len(existingRoleKeys) > 0 {
		resp.Diagnostics.AddError("The team already has custom roles assigned.", fmt.Sprintf("The team %q already has custom roles assigned.", teamKey))
	}

	customRoleKeys, sliceDiags := stringSliceFromSet(ctx, data.CustomRoleKeys)
	resp.Diagnostics.Append(sliceDiags...)

	desiredRoleAttrs, raDiags := roleAttributesFromFrameworkMap(ctx, data.RoleAttributes)
	resp.Diagnostics.Append(raDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	patchInstructions := make([]map[string]interface{}, 0)

	_, add := makeAddAndRemoveArrays([]string{}, customRoleKeys)
	if len(add) > 0 {
		instruction := map[string]interface{}{
			"kind":   "addCustomRoles",
			"values": add,
		}
		patchInstructions = append(patchInstructions, instruction)
	}

	if !data.RoleAttributes.IsNull() && !data.RoleAttributes.IsUnknown() {
		var existingRoleAttrs map[string][]string
		if team.RoleAttributes != nil {
			existingRoleAttrs = *team.RoleAttributes
		}
		patchInstructions = append(patchInstructions, diffRoleAttributePatches(existingRoleAttrs, desiredRoleAttrs)...)
	}

	if len(patchInstructions) > 0 {
		patch := ldapi.TeamPatchInput{
			Comment:      strPtr("Updated by Terraform"),
			Instructions: patchInstructions,
		}

		var err error
		err = r.client.withConcurrency(r.client.ctx, func() error {
			_, _, err = r.client.ld.TeamsApi.PatchTeam(r.client.ctx, teamKey).TeamPatchInput(patch).Execute()
			return err
		})
		if err != nil {
			resp.Diagnostics.AddError("Unable to update team custom roles", fmt.Sprintf("Unable to modify the %q team's custom roles. %s", teamKey, handleLdapiErr(err)))
			return
		}

		// Fetch all role keys with pagination to verify they were added
		returnedRoleKeys, err := getAllTeamCustomRoleKeys(r.client, teamKey)
		if err != nil {
			resp.Diagnostics.AddError("Unable to get team roles", fmt.Sprintf("Received an error when fetching roles for team %q: %s", teamKey, err))
			return
		}
		for _, role := range customRoleKeys {
			if !stringInSlice(role, returnedRoleKeys) {
				resp.Diagnostics.AddError("Unable to add custom role to team", fmt.Sprintf("Unable to add custom role with key %q to the team. Ensure the custom role exists first.", role))
			}
		}

		customRoleSet, diags := setFromStringSlice(ctx, returnedRoleKeys)
		resp.Diagnostics.Append(diags...)
		data.CustomRoleKeys = customRoleSet
	}

	// data.RoleAttributes already holds the plan value (or null). The
	// replaceRoleAttributes patch is wholesale, so the team now reflects exactly
	// what we sent. Avoid re-reading from the API here because LD may return
	// values in a different order than the plan, which would trip the
	// "Provider produced inconsistent result after apply" check.

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TeamRoleMappingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TeamRoleMappingResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamKey := data.TeamKey.ValueString()

	// First verify the team exists - this provides clearer error messages than
	// failing on the roles API call. Expand roleAttributes so we can populate them
	// in the same call.
	var team *ldapi.Team
	var res *http.Response
	var err error
	err = r.client.withConcurrency(r.client.ctx, func() error {
		team, res, err = r.client.ld404Retry.TeamsApi.GetTeam(r.client.ctx, teamKey).Expand("roleAttributes").Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			// Team was deleted outside of Terraform, remove from state
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to get team", fmt.Sprintf("Received an error when fetching the team %q: %s", teamKey, handleLdapiErr(err)))
		return
	}

	// Fetch all role keys with pagination using the 404-retry client
	// We use the ld404Retry API client because users that provision their team via Okta team sync may
	// see a delay before the team appears in LaunchDarkly. sc-218015
	roleKeys, err := getAllTeamCustomRoleKeysWithRetry(r.client, teamKey)
	if err != nil {
		resp.Diagnostics.AddError("Unable to get team roles", fmt.Sprintf("Received an error when fetching roles for team %q: %s", teamKey, err))
		return
	}

	data.TeamKey = types.StringValue(teamKey)
	data.ID = types.StringValue(teamKey)
	customRoleSet, diags := setFromStringSlice(ctx, roleKeys)
	resp.Diagnostics.Append(diags...)
	data.CustomRoleKeys = customRoleSet

	// Only refresh role_attributes from the API when this resource already owns
	// the field (prior state has a non-null value). When state is null we treat
	// the field as unmanaged here — leaving SCIM-set or launchdarkly_team-managed
	// attributes alone instead of pulling them into our state and creating churn.
	if !data.RoleAttributes.IsNull() {
		priorRaw, priorDiags := roleAttributesFromFrameworkMap(ctx, data.RoleAttributes)
		resp.Diagnostics.Append(priorDiags...)
		var apiRaw map[string][]string
		if team.RoleAttributes != nil {
			apiRaw = *team.RoleAttributes
		}
		// Preserve the state's value ordering when the content is equal
		// unordered. Avoids spurious plan diffs when LD returns values in a
		// different order than the user wrote.
		if !roleAttributesEqual(priorRaw, apiRaw) {
			roleAttrsMap, raDiags := roleAttributesToFrameworkMap(team.RoleAttributes)
			resp.Diagnostics.Append(raDiags...)
			data.RoleAttributes = roleAttrsMap
		}
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TeamRoleMappingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *TeamRoleMappingResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read prior state so we can tell whether this resource already owned
	// role_attributes. Transitions:
	//   prior null + plan null  → not owned, do not touch
	//   prior null + plan set   → opting in, write
	//   prior set  + plan set   → owned, diff and patch
	//   prior set  + plan null  → opting out, clear team attributes
	var prior TeamRoleMappingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamKey := data.TeamKey.ValueString()

	// First verify the team exists - this provides clearer error messages than
	// failing on the roles API call. Expand roleAttributes so we can diff against
	// the API-current value before issuing patches.
	var team *ldapi.Team
	var res *http.Response
	var err error
	err = r.client.withConcurrency(r.client.ctx, func() error {
		team, res, err = r.client.ld404Retry.TeamsApi.GetTeam(r.client.ctx, teamKey).Expand("roleAttributes").Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			resp.Diagnostics.AddError("Team not found", fmt.Sprintf("Unable to update the team/role mapping because the team %q does not exist.", teamKey))
			return
		}
		resp.Diagnostics.AddError("Unable to get team", fmt.Sprintf("Received an error when fetching the team %q: %s", teamKey, handleLdapiErr(err)))
		return
	}

	// Fetch existing role keys with pagination
	existingRoleKeys, err := getAllTeamCustomRoleKeysWithRetry(r.client, teamKey)
	if err != nil {
		resp.Diagnostics.AddError("Unable to get team roles", fmt.Sprintf("Received an error when fetching roles for team %q: %s", teamKey, err))
		return
	}

	customRoleKeys, sliceDiags := stringSliceFromSet(ctx, data.CustomRoleKeys)
	resp.Diagnostics.Append(sliceDiags...)

	desiredRoleAttrs, raDiags := roleAttributesFromFrameworkMap(ctx, data.RoleAttributes)
	resp.Diagnostics.Append(raDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	patchInstructions := make([]map[string]interface{}, 0)

	remove, add := makeAddAndRemoveArrays(existingRoleKeys, customRoleKeys)
	if len(add) > 0 {
		instruction := map[string]interface{}{
			"kind":   "addCustomRoles",
			"values": add,
		}
		patchInstructions = append(patchInstructions, instruction)
	}

	if len(remove) > 0 {
		instruction := map[string]interface{}{
			"kind":   "removeCustomRoles",
			"values": remove,
		}
		patchInstructions = append(patchInstructions, instruction)
	}

	// Role-attributes ownership transitions, see comment at top of Update.
	priorOwned := !prior.RoleAttributes.IsNull() && !prior.RoleAttributes.IsUnknown()
	planOwned := !data.RoleAttributes.IsNull() && !data.RoleAttributes.IsUnknown()
	var existingRoleAttrs map[string][]string
	if team.RoleAttributes != nil {
		existingRoleAttrs = *team.RoleAttributes
	}
	switch {
	case planOwned:
		patchInstructions = append(patchInstructions, diffRoleAttributePatches(existingRoleAttrs, desiredRoleAttrs)...)
	case priorOwned && !planOwned:
		// User removed role_attributes from configuration → clear team-side
		// values that this resource previously wrote.
		patchInstructions = append(patchInstructions, diffRoleAttributePatches(existingRoleAttrs, nil)...)
	}

	if len(patchInstructions) > 0 {
		patch := ldapi.TeamPatchInput{
			Comment:      strPtr("Updated by Terraform"),
			Instructions: patchInstructions,
		}

		var err error
		err = r.client.withConcurrency(r.client.ctx, func() error {
			_, _, err = r.client.ld.TeamsApi.PatchTeam(r.client.ctx, teamKey).TeamPatchInput(patch).Execute()
			return err
		})
		if err != nil {
			resp.Diagnostics.AddError("Unable to update team custom roles", fmt.Sprintf("Unable to modify the %q team's custom roles. %s", teamKey, handleLdapiErr(err)))
			return
		}

		// Fetch all role keys with pagination to verify they were updated
		returnedRoleKeys, err := getAllTeamCustomRoleKeys(r.client, teamKey)
		if err != nil {
			resp.Diagnostics.AddError("Unable to get team roles", fmt.Sprintf("Received an error when fetching roles for team %q: %s", teamKey, err))
			return
		}

		data.TeamKey = types.StringValue(teamKey)
		data.ID = types.StringValue(teamKey)
		for _, role := range customRoleKeys {
			if !stringInSlice(role, returnedRoleKeys) {
				resp.Diagnostics.AddError("Unable to add custom role to team", fmt.Sprintf("Unable to add custom role with key %q to the team. Ensure the custom role exists first.", role))
			}
		}

		customRoleSet, diags := setFromStringSlice(ctx, returnedRoleKeys)
		resp.Diagnostics.Append(diags...)
		data.CustomRoleKeys = customRoleSet
	}

	// data.RoleAttributes already holds the plan value (or null for opt-out).
	// We deliberately do not re-read it from the API: LD may return values in
	// a different order than the plan, which would trigger Terraform's
	// "Provider produced inconsistent result after apply" check.

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TeamRoleMappingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *TeamRoleMappingResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamKey := data.TeamKey.ValueString()
	customRoleKeys, sliceDiags := stringSliceFromSet(ctx, data.CustomRoleKeys)
	resp.Diagnostics.Append(sliceDiags...)

	instructions := make([]map[string]interface{}, 0)

	if len(customRoleKeys) > 0 {
		instructions = append(instructions, map[string]interface{}{
			"kind":   "removeCustomRoles",
			"values": customRoleKeys,
		})
	}

	// If this resource owned role_attributes (state has a non-null value), clear
	// them on the team so destroy doesn't leave stale scopes behind. When state
	// is null we never wrote them, so leave the team alone.
	if !data.RoleAttributes.IsNull() && !data.RoleAttributes.IsUnknown() {
		instructions = append(instructions, map[string]interface{}{
			"kind":  "replaceRoleAttributes",
			"value": map[string][]string{},
		})
	}

	if len(instructions) > 0 {
		patch := ldapi.TeamPatchInput{
			Comment:      strPtr("Updated by Terraform"),
			Instructions: instructions,
		}

		var err error
		err = r.client.withConcurrency(r.client.ctx, func() error {
			_, _, err = r.client.ld.TeamsApi.PatchTeam(r.client.ctx, teamKey).TeamPatchInput(patch).Execute()
			return err
		})
		if err != nil {
			resp.Diagnostics.AddError("Unable to delete team custom roles", fmt.Sprintf("Unable to modify the %q team's custom roles. %s", teamKey, handleLdapiErr(err)))
			return
		}
	}
}

func (r *TeamRoleMappingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("team_key"), req, resp)
}

func NewTeamRoleMappingResource() resource.Resource {
	return &TeamRoleMappingResource{}
}
