package launchdarkly

import (
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
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
			State: resourceTeamMemberImport,
		},

		Schema: map[string]*schema.Schema{
			EMAIL: {
				Type:     schema.TypeString,
				Required: true,
			},
			FIRST_NAME: {
				Type:     schema.TypeString,
				Optional: true,
			},
			LAST_NAME: {
				Type:     schema.TypeString,
				Optional: true,
			},
			ROLE: {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateTeamMemberRole,
			},
			CUSTOM_ROLES: {
				Type:     schema.TypeSet,
				Set:      schema.HashString,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
		},
	}
}

func validateTeamMemberRole(val interface{}, key string) (warns []string, errs []error) {
	v := val.(string)
	switch v {
	case "reader", "writer", "admin":
		// Do nothing
	default:
		errs = append(errs, fmt.Errorf("%q must be either `reader`, `writer`, or `admin`. Got: %s", key, v))
	}
	return warns, errs
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

	members, _, err := client.ld.TeamMembersApi.PostMembers(client.ctx, []ldapi.MembersBody{membersBody})
	if err != nil {
		return fmt.Errorf("failed to create team member with email: %s: %v", memberEmail, handleLdapiErr(err))
	}

	d.SetId(members.Items[0].Id)
	return resourceTeamMemberRead(d, metaRaw)
}

func resourceTeamMemberRead(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	memberID := d.Id()

	member, res, err := client.ld.TeamMembersApi.GetMember(client.ctx, memberID)
	if isStatusNotFound(res) {
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
	err = d.Set(CUSTOM_ROLES, member.CustomRoles)
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

	patch := []ldapi.PatchOperation{
		// these are the only fields we are allowed to update:
		patchReplace("/role", &memberRole),
		patchReplace("/customRoles", &customRolesRaw),
	}

	_, _, err := repeatUntilNoConflict(func() (interface{}, *http.Response, error) {
		return client.ld.TeamMembersApi.PatchMember(client.ctx, memberID, patch)
	})
	if err != nil {
		return fmt.Errorf("failed to update team member with id %q: %s", memberID, handleLdapiErr(err))
	}

	return resourceTeamMemberRead(d, metaRaw)
}

func resourceTeamMemberDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)

	_, err := client.ld.TeamMembersApi.DeleteMember(client.ctx, d.Id())
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

func resourceTeamMemberImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if err := resourceTeamMemberRead(d, meta); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}
