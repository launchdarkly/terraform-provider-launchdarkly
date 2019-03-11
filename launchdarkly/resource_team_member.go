package launchdarkly

import (
	"fmt"

	"github.com/launchdarkly/api-client-go"

	"github.com/hashicorp/terraform/helper/schema"
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
			_id: &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			email: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			first_name: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			last_name: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			role: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			custom_roles: &schema.Schema{
				Type:     schema.TypeSet,
				Set:      stringHash,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
		},
	}
}

func resourceTeamMemberCreate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	memberEmail := d.Get(email).(string)
	firstName := d.Get(first_name).(string)
	lastName := d.Get(last_name).(string)
	memberRole := ldapi.Role(d.Get(role).(string))
	customRolesRaw := d.Get(custom_roles).(*schema.Set).List()

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

	members, _, err := client.LaunchDarkly.TeamMembersApi.PostMembers(client.Ctx, []ldapi.MembersBody{membersBody})
	if err != nil {
		if swaggerErr, ok := err.(ldapi.GenericSwaggerError); ok {
			return fmt.Errorf("failed to create team member with email: %s: %v %s", memberEmail, swaggerErr.Error(), swaggerErr.Body())
		}

		return fmt.Errorf("failed to create team member with email: %s: %v", memberEmail, err)
	}

	d.SetId(members.Items[0].Id)
	return resourceTeamMemberRead(d, metaRaw)
}

func resourceTeamMemberRead(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	memberId := d.Id()

	member, _, err := client.LaunchDarkly.TeamMembersApi.GetMember(client.Ctx, memberId)
	if err != nil {
		return fmt.Errorf("failed to get member with id %q: %v", memberId, err)
	}

	d.SetId(member.Id)
	d.Set(_id, member.Id)
	d.Set(email, member.Email)
	d.Set(first_name, member.FirstName)
	d.Set(last_name, member.LastName)
	d.Set(role, member.Role)
	d.Set(custom_roles, member.CustomRoles)

	return nil
}

func resourceTeamMemberUpdate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	memberId := d.Id()
	firstName := d.Get(first_name).(string)
	lastName := d.Get(last_name).(string)
	memberRole := d.Get(role).(ldapi.Role)
	customRolesRaw := d.Get(custom_roles).(*schema.Set).List()

	patch := []ldapi.PatchOperation{
		patchReplace("/firstName", &firstName),
		patchReplace("/lastName", &lastName),
		patchReplace("/role", &memberRole),
		patchReplace("/customRoles", &customRolesRaw),
	}

	_, _, err := client.LaunchDarkly.TeamMembersApi.PatchMember(client.Ctx, memberId, patch)
	if err != nil {
		return fmt.Errorf("failed to update team member with id %q: %s", memberId, err)
	}

	return resourceTeamMemberRead(d, metaRaw)
}

func resourceTeamMemberDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)

	_, err := client.LaunchDarkly.TeamMembersApi.DeleteMember(client.Ctx, d.Id())
	if err != nil {
		return fmt.Errorf("failed to delete team member with id %q: %s", d.Id(), err)
	}

	return nil
}

func resourceTeamMemberExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	return teamMemberExists(d.Id(), metaRaw.(*Client))
}

func teamMemberExists(memberId string, meta *Client) (bool, error) {
	_, httpResponse, err := meta.LaunchDarkly.TeamMembersApi.GetMember(meta.Ctx, memberId)
	if httpResponse != nil && httpResponse.StatusCode == 404 {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get team member with id %q: %v", memberId, err)
	}

	return true, nil
}

func resourceTeamMemberImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	d.SetId(d.Id())
	d.Set(_id, d.Id())

	if err := resourceTeamMemberRead(d, meta); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}
