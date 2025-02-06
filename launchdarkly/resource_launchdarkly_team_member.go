package launchdarkly

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	ldapi "github.com/launchdarkly/api-client-go/v17"
)

func resourceTeamMember() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceTeamMemberCreate,
		ReadContext:   resourceTeamMemberRead,
		UpdateContext: resourceTeamMemberUpdate,
		DeleteContext: resourceTeamMemberDelete,
		Exists:        resourceTeamMemberExists,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			ID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The 24 character alphanumeric ID of the team member.",
			},
			EMAIL: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: addForceNewDescription("The unique email address associated with the team member.", true),
			},
			FIRST_NAME: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The team member's given name. Once created, this cannot be updated except by the team member.",
			},
			LAST_NAME: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "TThe team member's family name. Once created, this cannot be updated except by the team member.",
			},
			ROLE: {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				Description:      "The role associated with team member. Supported roles are `reader`, `writer`, `no_access`, or `admin`. If you don't specify a role, `reader` is assigned by default.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"reader", "writer", "admin", "no_access"}, false)),
				AtLeastOneOf:     []string{ROLE, CUSTOM_ROLES},
			},
			CUSTOM_ROLES: {
				Type:         schema.TypeSet,
				Set:          schema.HashString,
				Elem:         &schema.Schema{Type: schema.TypeString},
				Optional:     true,
				Description:  "The list of custom roles keys associated with the team member. Custom roles are only available to customers on an Enterprise plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).\n\n-> **Note:** each `launchdarkly_team_member` must have either a `role` or `custom_roles` argument.",
				AtLeastOneOf: []string{ROLE, CUSTOM_ROLES},
			},
			ROLE_ATTRIBUTES: roleAttributesSchema(false),
		},

		Description: `Provides a LaunchDarkly team member resource.

This resource allows you to create and manage team members within your LaunchDarkly organization.

-> **Note:** You can only manage team members with "admin" level personal access tokens. To learn more, read [Managing Teams](https://docs.launchdarkly.com/home/teams/managing).`,
	}
}

func resourceTeamMemberCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	memberEmail := d.Get(EMAIL).(string)
	firstName := d.Get(FIRST_NAME).(string)
	lastName := d.Get(LAST_NAME).(string)
	memberRole := d.Get(ROLE).(string)
	customRolesRaw := d.Get(CUSTOM_ROLES).(*schema.Set).List()

	customRoles := make([]string, len(customRolesRaw))
	for i, cr := range customRolesRaw {
		customRoles[i] = cr.(string)
	}

	membersBody := ldapi.NewMemberForm{
		Email:       memberEmail,
		FirstName:   &firstName,
		LastName:    &lastName,
		Role:        &memberRole,
		CustomRoles: customRoles,
	}

	members, _, err := client.ld.AccountMembersApi.PostMembers(client.ctx).NewMemberForm([]ldapi.NewMemberForm{membersBody}).Execute()
	if err != nil {
		return diag.Errorf("failed to create team member with email: %s: %v", memberEmail, handleLdapiErr(err))
	}

	d.SetId(members.Items[0].Id)
	return resourceTeamMemberRead(ctx, d, metaRaw)
}

func resourceTeamMemberRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)
	memberID := d.Id()

	member, res, err := client.ld.AccountMembersApi.GetMember(client.ctx, memberID).Execute()
	if isStatusNotFound(res) {
		log.Printf("[WARN] failed to find member with id %q, removing from state", memberID)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("[WARN] failed to find member with id %q, removing from state", memberID),
		})
		d.SetId("")
		return diags
	}
	if err != nil {
		return diag.Errorf("failed to get member with id %q: %v", memberID, err)
	}

	d.SetId(member.Id)
	_ = d.Set(EMAIL, member.Email)
	_ = d.Set(FIRST_NAME, member.FirstName)
	_ = d.Set(LAST_NAME, member.LastName)
	_ = d.Set(ROLE, member.Role)

	customRoleKeys, err := customRoleIDsToKeys(client, member.CustomRoles)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set(CUSTOM_ROLES, customRoleKeys)
	if err != nil {
		return diag.Errorf("failed to set custom roles on team member with id %q: %v", member.Id, err)
	}
	return diags
}

func resourceTeamMemberUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	memberID := d.Id()
	memberRole := d.Get(ROLE).(string)
	customRolesRaw := d.Get(CUSTOM_ROLES).(*schema.Set).List()

	customRoleKeys := make([]string, len(customRolesRaw))
	for i, cr := range customRolesRaw {
		customRoleKeys[i] = cr.(string)
	}
	customRoleIds, err := customRoleKeysToIDs(client, customRoleKeys)
	if err != nil {
		return diag.FromErr(err)
	}

	patch := []ldapi.PatchOperation{
		// these are the only fields we are allowed to update:
		patchReplace("/role", &memberRole),
		patchReplace("/customRoles", &customRoleIds),
	}

	_, _, err = client.ld.AccountMembersApi.PatchMember(client.ctx, memberID).PatchOperation(patch).Execute()
	if err != nil {
		return diag.Errorf("failed to update team member with id %q: %s", memberID, handleLdapiErr(err))
	}

	return resourceTeamMemberRead(ctx, d, metaRaw)
}

func resourceTeamMemberDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)

	_, err := client.ld.AccountMembersApi.DeleteMember(client.ctx, d.Id()).Execute()
	if err != nil {
		return diag.Errorf("failed to delete team member with id %q: %s", d.Id(), handleLdapiErr(err))
	}

	return diags
}

func resourceTeamMemberExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	return teamMemberExists(d.Id(), metaRaw.(*Client))
}

func teamMemberExists(memberID string, meta *Client) (bool, error) {
	_, res, err := meta.ld.AccountMembersApi.GetMember(meta.ctx, memberID).Execute()
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get team member with id %q: %v", memberID, handleLdapiErr(err))
	}

	return true, nil
}
