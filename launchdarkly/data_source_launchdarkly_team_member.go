package launchdarkly

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v10"
)

func memberSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		EMAIL: {
			Type:     schema.TypeString,
			Required: true,
		},
		ID: {
			Type:     schema.TypeString,
			Computed: true,
			Optional: true,
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
	}
}

func dataSourceTeamMember() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceTeamMemberRead,
		Schema:      memberSchema(),
	}
}

func getTeamMemberByEmail(client *Client, memberEmail string) (*ldapi.Member, error) {
	// this should be the max limit allowed when the member-list-max-limit flag is on
	teamMemberLimit := int64(1000)

	// After changing this to query by member email, we shouldn't need the limit and recursion on requests, but leaving it in just to be extra safe
	members, _, err := client.ld.AccountMembersApi.GetMembers(client.ctx).Filter(fmt.Sprintf("query:%s", url.QueryEscape(memberEmail))).Execute()

	if err != nil {
		return nil, fmt.Errorf("failed to read team member with email: %s: %v", memberEmail, handleLdapiErr(err))
	}

	totalMemberCount := int(*members.TotalCount)

	memberItems := members.Items
	membersPulled := len(memberItems)
	for membersPulled < totalMemberCount {
		offset := int64(membersPulled)
		newMembers, _, err := client.ld.AccountMembersApi.GetMembers(client.ctx).Limit(teamMemberLimit).Offset(offset).Filter(fmt.Sprintf("query:%s", url.QueryEscape(memberEmail))).Execute()

		if err != nil {
			return nil, fmt.Errorf("failed to read team member with email: %s: %v", memberEmail, handleLdapiErr(err))
		}

		memberItems = append(memberItems, newMembers.Items...)
		membersPulled = len(memberItems)
	}

	for _, member := range memberItems {
		if member.Email == memberEmail {
			return &member, nil
		}
	}
	return nil, fmt.Errorf("failed to find team member with email: %s", memberEmail)

}

func dataSourceTeamMemberRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*Client)
	memberEmail := d.Get(EMAIL).(string)
	member, err := getTeamMemberByEmail(client, memberEmail)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(member.Id)
	_ = d.Set(EMAIL, member.Email)
	_ = d.Set(FIRST_NAME, member.FirstName)
	_ = d.Set(LAST_NAME, member.LastName)
	_ = d.Set(ROLE, member.Role)
	err = d.Set(CUSTOM_ROLES, member.CustomRoles)
	if err != nil {
		return diag.Errorf("failed to set custom roles on team member with email %q: %v", member.Email, err)
	}

	return diags
}
