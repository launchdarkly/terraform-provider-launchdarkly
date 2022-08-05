package launchdarkly

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	ldapi "github.com/launchdarkly/api-client-go/v10"
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
			EMAIL: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The team member's email address",
			},
			FIRST_NAME: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The team member's first name",
			},
			LAST_NAME: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The team member's last name",
			},
			ROLE: {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				Description:      "The team member's role. This must be reader, writer, admin, or no_access. Team members must have either a role or custom role",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"reader", "writer", "admin", "no_access"}, false)),
				AtLeastOneOf:     []string{ROLE, CUSTOM_ROLES},
			},
			CUSTOM_ROLES: {
				Type:         schema.TypeSet,
				Set:          schema.HashString,
				Elem:         &schema.Schema{Type: schema.TypeString},
				Optional:     true,
				Description:  "IDs or keys of custom roles. Team members must have either a role or custom role",
				AtLeastOneOf: []string{ROLE, CUSTOM_ROLES},
			},
		},
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
