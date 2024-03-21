package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v15"
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
			// In SDKv2, resources and data sources automatically included an implicit, root level id attribute.
			// In the framework, the id attribute is not implicitly added.
			// Conventionally, id is a computed attribute that contains the identifier for the resource.
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
	// Prevent panic if the provider has not been configured
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData))
	}

	r.client = client
}

func (r *TeamRoleMappingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *TeamRoleMappingResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	teamKey := data.TeamKey.ValueString()
	team, res, err := r.client.ld404Retry.TeamsApi.GetTeam(r.client.ctx, teamKey).Expand("roles").Execute()
	if err != nil {
		if isStatusNotFound(res) {
			resp.Diagnostics.AddError("Team not found", fmt.Sprintf("Unable to create the team/role mapping because the team %q does not exist.", teamKey))
			return
		}

		resp.Diagnostics.AddError("Unable to get team", fmt.Sprintf("Received an error when fetching the team %q: %s", teamKey, handleLdapiErr((err))))
		return
	}

	if team.Roles != nil && len(team.Roles.Items) > 0 {
		resp.Diagnostics.AddError("The team already has custom roles assigned.", fmt.Sprintf("The team %q already has custom roles assigned.", teamKey))
	}

	customRoleKeys := make([]string, 0, len(data.CustomRoleKeys.Elements()))
	resp.Diagnostics.Append(data.CustomRoleKeys.ElementsAs(ctx, &customRoleKeys, false)...)

	patchInstructions := make([]map[string]interface{}, 0)

	_, add := makeAddAndRemoveArrays([]string{}, customRoleKeys)
	if len(add) > 0 {
		instruction := map[string]interface{}{
			"kind":   "addCustomRoles",
			"values": add,
		}
		patchInstructions = append(patchInstructions, instruction)
	}

	if len(patchInstructions) > 0 {
		patch := ldapi.TeamPatchInput{
			Comment:      strPtr("Updated by Terraform"),
			Instructions: patchInstructions,
		}

		_, _, err := r.client.ld.TeamsApi.PatchTeam(r.client.ctx, teamKey).TeamPatchInput(patch).Execute()
		if err != nil {
			resp.Diagnostics.AddError("Unable to update team custom roles", fmt.Sprintf("Unable to modify the %q team's custom roles. %s", teamKey, handleLdapiErr(err)))
			return
		}

		team, res, err := r.client.ld.TeamsApi.GetTeam(r.client.ctx, teamKey).Expand("roles").Execute()
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

		returnedRoleKeys := make([]string, 0, len(team.Roles.Items))
		for _, role := range team.Roles.Items {
			returnedRoleKeys = append(returnedRoleKeys, role.GetKey())
		}
		for _, role := range customRoleKeys {
			if !stringInSlice(role, returnedRoleKeys) {
				resp.Diagnostics.AddError("Unable to add custom role to team", fmt.Sprintf("Unable to add custom role with key %q to the team. Ensure the custom role exists first.", role))
			}
		}

		customRoleSet, diags := types.SetValueFrom(ctx, types.StringType, returnedRoleKeys)
		resp.Diagnostics.Append(diags...)
		data.CustomRoleKeys = customRoleSet
	}

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

	// we use the ld404Retry API client to fetch the team because users that provision their team via Okta team sync may
	// see a delay before the team appears in LaunchDarkly. sc-218015
	team, res, err := r.client.ld404Retry.TeamsApi.GetTeam(r.client.ctx, teamKey).Expand("roles").Execute()
	if err != nil {
		if isStatusNotFound(res) {
			resp.Diagnostics.AddError("Team not found", fmt.Sprintf("Unable to read the team/role mapping because the team %q does not exist.", teamKey))
			return
		}

		resp.Diagnostics.AddError("Unable to get team", fmt.Sprintf("Received an error when fetching the team %q: %s", teamKey, handleLdapiErr((err))))
		return
	}

	data.TeamKey = types.StringValue(*team.Key)
	data.ID = types.StringValue(*team.Key)
	coercedRoleKeys := make([]attr.Value, 0, len(team.Roles.Items))
	for _, customRole := range team.Roles.Items {
		coercedRoleKeys = append(coercedRoleKeys, types.StringValue(customRole.GetKey()))
	}
	customRoleSet, diags := types.SetValue(types.StringType, coercedRoleKeys)
	resp.Diagnostics.Append(diags...)
	data.CustomRoleKeys = customRoleSet

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

	teamKey := data.TeamKey.ValueString()
	team, res, err := r.client.ld404Retry.TeamsApi.GetTeam(r.client.ctx, teamKey).Expand("roles").Execute()
	if err != nil {
		if isStatusNotFound(res) {
			resp.Diagnostics.AddError("Team not found", fmt.Sprintf("Unable to get the team/role mapping because the team %q does not exist.", teamKey))
			return
		}

		resp.Diagnostics.AddError("Unable to get team", fmt.Sprintf("Received an error when fetching the team %q: %s", teamKey, handleLdapiErr((err))))
		return
	}

	existingCustomRoles := team.Roles.Items
	existingRoleKeys := make([]string, 0, len(existingCustomRoles))
	for _, role := range existingCustomRoles {
		existingRoleKeys = append(existingRoleKeys, role.GetKey())
	}

	customRoleKeys := make([]string, 0, len(data.CustomRoleKeys.Elements()))
	resp.Diagnostics.Append(data.CustomRoleKeys.ElementsAs(ctx, &customRoleKeys, false)...)

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

	if len(patchInstructions) > 0 {
		patch := ldapi.TeamPatchInput{
			Comment:      strPtr("Updated by Terraform"),
			Instructions: patchInstructions,
		}

		_, _, err := r.client.ld.TeamsApi.PatchTeam(r.client.ctx, teamKey).TeamPatchInput(patch).Execute()
		if err != nil {
			resp.Diagnostics.AddError("Unable to update team custom roles", fmt.Sprintf("Unable to modify the %q team's custom roles. %s", teamKey, handleLdapiErr(err)))
			return
		}

		// We need to fetch the team again via GET (with the expand=roles query param) because the PATCH response does not
		// include custom role information.
		team, res, err := r.client.ld.TeamsApi.GetTeam(r.client.ctx, teamKey).Expand("roles").Execute()
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
		returnedRoleKeys := make([]string, 0, len(team.Roles.Items))
		for _, role := range team.Roles.Items {
			returnedRoleKeys = append(returnedRoleKeys, role.GetKey())
		}
		for _, role := range customRoleKeys {
			if !stringInSlice(role, returnedRoleKeys) {
				resp.Diagnostics.AddError("Unable to add custom role to team", fmt.Sprintf("Unable to add custom role with key %q to the team. Ensure the custom role exists first.", role))
			}
		}

		customRoleSet, diags := types.SetValueFrom(ctx, types.StringType, returnedRoleKeys)
		resp.Diagnostics.Append(diags...)
		data.CustomRoleKeys = customRoleSet
	}

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
	customRoleKeys := make([]string, 0, len(data.CustomRoleKeys.Elements()))
	resp.Diagnostics.Append(data.CustomRoleKeys.ElementsAs(ctx, &customRoleKeys, false)...)

	if len(customRoleKeys) > 0 {
		instructions := []map[string]interface{}{
			{
				"kind":   "removeCustomRoles",
				"values": customRoleKeys,
			},
		}

		patch := ldapi.TeamPatchInput{
			Comment:      strPtr("Updated by Terraform"),
			Instructions: instructions,
		}

		_, _, err := r.client.ld.TeamsApi.PatchTeam(r.client.ctx, teamKey).TeamPatchInput(patch).Execute()
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
