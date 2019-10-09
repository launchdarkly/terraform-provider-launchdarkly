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
			email: {
				Type:     schema.TypeString,
				Required: true,
			},
			first_name: {
				Type:     schema.TypeString,
				Computed: true,
			},
			last_name: {
				Type:     schema.TypeString,
				Computed: true,
			},
			role: {
				Type:     schema.TypeString,
				Computed: true,
			},
			custom_roles: {
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
	memberEmail := d.Get(email).(string)
	member, err := getTeamMemberByEmail(client, memberEmail)
	if err != nil {
		return err
	}
	d.SetId(member.Id)
	_ = d.Set(email, member.Email)
	_ = d.Set(first_name, member.FirstName)
	_ = d.Set(last_name, member.LastName)
	_ = d.Set(role, member.Role)
	err = d.Set(custom_roles, member.CustomRoles)
	if err != nil {
		return fmt.Errorf("failed to set custom roles on team member with email %q: %v", member.Email, err)
	}

	return nil
}
