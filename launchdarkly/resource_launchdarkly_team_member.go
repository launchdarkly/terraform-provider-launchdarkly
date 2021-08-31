package launchdarkly

import (
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	ldapi "github.com/launchdarkly/api-client-go"
)

func resourceTeamMember() *schema.Resource {
	return &schema.Resource{
		Create: resourceTeamMemberCreate,
		Read:   resourceTeamMemberRead,
		Update: resourceTeamMemberUpdate,
		Delete: resourceTeamMemberDelete,
		Exists: resourceTeamMemberExists,

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
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The team member's role. This must be reader, writer, admin, or owner. Team members must have either a role or custom role",
				ValidateFunc: validation.StringInSlice([]string{"reader", "writer", "admin"}, false),
				AtLeastOneOf: []string{ROLE, CUSTOM_ROLES},
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

func resourceTeamMemberCreate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	memberEmail := d.Get(EMAIL).(string)
	firstName := d.Get(FIRST_NAME).(string)
	lastName := d.Get(LAST_NAME).(string)
	memberRole := ldapi.Role(d.Get(ROLE).(string))
	customRolesRaw := d.Get(CUSTOM_ROLES).(*schema.Set).List()

	customRoles := make([]string, len(customRolesRaw))
	for i, cr := range customRolesRaw {
		customRoles[i] = cr.(string)
	}

	membersBody := ldapi.MembersBody{
		Email:       memberEmail,
		FirstName:   firstName,
		LastName:    lastName,
		Role:        &memberRole,
		CustomRoles: customRoles,
	}

	membersRaw, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.TeamMembersApi.PostMembers(client.ctx, []ldapi.MembersBody{membersBody})
	})
	members := membersRaw.(ldapi.Members)
	if err != nil {
		return fmt.Errorf("failed to create team member with email: %s: %v", memberEmail, handleLdapiErr(err))
	}

	d.SetId(members.Items[0].Id)
	return resourceTeamMemberRead(d, metaRaw)
}

func resourceTeamMemberRead(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	memberID := d.Id()

	memberRaw, res, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.TeamMembersApi.GetMember(client.ctx, memberID)
	})
	member := memberRaw.(ldapi.Member)
	if isStatusNotFound(res) {
		log.Printf("[WARN] failed to find member with id %q, removing from state", memberID)
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get member with id %q: %v", memberID, err)
	}

	d.SetId(member.Id)
	_ = d.Set(EMAIL, member.Email)
	_ = d.Set(FIRST_NAME, member.FirstName)
	_ = d.Set(LAST_NAME, member.LastName)
	_ = d.Set(ROLE, member.Role)

	customRoleKeys, err := customRoleIDsToKeys(client, member.CustomRoles)
	if err != nil {
		return err
	}
	err = d.Set(CUSTOM_ROLES, customRoleKeys)
	if err != nil {
		return fmt.Errorf("failed to set custom roles on team member with id %q: %v", member.Id, err)
	}
	return nil
}

func resourceTeamMemberUpdate(d *schema.ResourceData, metaRaw interface{}) error {
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
		return err
	}

	patch := []ldapi.PatchOperation{
		// these are the only fields we are allowed to update:
		patchReplace("/role", &memberRole),
		patchReplace("/customRoles", &customRoleIds),
	}

	_, _, err = handleRateLimit(func() (interface{}, *http.Response, error) {
		return handleNoConflict(func() (interface{}, *http.Response, error) {
			return client.ld.TeamMembersApi.PatchMember(client.ctx, memberID, patch)
		})
	})
	if err != nil {
		return fmt.Errorf("failed to update team member with id %q: %s", memberID, handleLdapiErr(err))
	}

	return resourceTeamMemberRead(d, metaRaw)
}

func resourceTeamMemberDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)

	_, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		res, err := client.ld.TeamMembersApi.DeleteMember(client.ctx, d.Id())
		return nil, res, err
	})
	if err != nil {
		return fmt.Errorf("failed to delete team member with id %q: %s", d.Id(), handleLdapiErr(err))
	}

	return nil
}

func resourceTeamMemberExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	return teamMemberExists(d.Id(), metaRaw.(*Client))
}

func teamMemberExists(memberID string, meta *Client) (bool, error) {
	_, res, err := meta.ld.TeamMembersApi.GetMember(meta.ctx, memberID)
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get team member with id %q: %v", memberID, handleLdapiErr(err))
	}

	return true, nil
}
