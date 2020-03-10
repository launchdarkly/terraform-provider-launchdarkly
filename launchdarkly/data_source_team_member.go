package launchdarkly

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
)

func dataSourceTeamMember() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceTeamMemberRead,

		Schema: map[string]*schema.Schema{
			EMAIL: {
				Type:     schema.TypeString,
				Required: true,
			},
			FIRST_NAME: {
				Type:     schema.TypeString,
				Computed: true,
			},
			LAST_NAME: {
				Type:     schema.TypeString,
				Computed: true,
			},
			ROLE: {
				Type:     schema.TypeString,
				Computed: true,
			},
			CUSTOM_ROLES: {
				Type:     schema.TypeSet,
				Set:      schema.HashString,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
		},
	}
}

func getTeamMemberByEmail(client *Client, memberEmail string) (*ldapi.Member, error) {
	members, _, err := client.ld.TeamMembersApi.GetMembers(client.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read team member with email: %s: %v", memberEmail, handleLdapiErr(err))
	}
	for _, member := range members.Items {
		if member.Email == memberEmail {
			return &member, nil
		}
	}
	return nil, fmt.Errorf("failed to find team member with email: %s", memberEmail)

}

func dataSourceTeamMemberRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)
	memberEmail := d.Get(EMAIL).(string)
	member, err := getTeamMemberByEmail(client, memberEmail)
	if err != nil {
		return err
	}
	d.SetId(member.Id)
	_ = d.Set(EMAIL, member.Email)
	_ = d.Set(FIRST_NAME, member.FirstName)
	_ = d.Set(LAST_NAME, member.LastName)
	_ = d.Set(ROLE, member.Role)
	err = d.Set(CUSTOM_ROLES, member.CustomRoles)
	if err != nil {
		return fmt.Errorf("failed to set custom roles on team member with email %q: %v", member.Email, err)
	}

	return nil
}
