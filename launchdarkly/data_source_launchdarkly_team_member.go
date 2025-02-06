package launchdarkly

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v17"
)

func memberSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		EMAIL: {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The unique email address associated with the team member.",
		},
		ID: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The 24 character alphanumeric ID of the team member.",
		},
		FIRST_NAME: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The team member's given name.",
		},
		LAST_NAME: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The team member's family name.",
		},
		ROLE: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The role associated with team member. Possible roles are `owner`, `reader`, `writer`, or `admin`.",
		},
		CUSTOM_ROLES: {
			Type:        schema.TypeSet,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Computed:    true,
			Description: `The list of custom roles keys associated with the team member. Custom roles are only available to customers on an Enterprise plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).`,
		},
		ROLE_ATTRIBUTES: roleAttributesSchema(true),
	}
}

func dataSourceTeamMember() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceTeamMemberRead,
		Schema:      memberSchema(),

		Description: `Provides a LaunchDarkly team member data source.

This data source allows you to retrieve team member information from your LaunchDarkly organization.`,
	}
}

func getTeamMemberByEmail(client *Client, memberEmail string) (*ldapi.Member, error) {
	// this should be the max limit allowed when the member-list-max-limit flag is on
	teamMemberLimit := int64(1000)

	// After changing this to query by member email, we shouldn't need the limit and recursion on requests, but leaving it in just to be extra safe
	members, _, err := client.ld.AccountMembersApi.GetMembers(client.ctx).Filter(fmt.Sprintf("query:%s", url.QueryEscape(memberEmail))).Expand("roleAttributes").Execute()

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
	err = d.Set(ROLE_ATTRIBUTES, roleAttributesToResourceData(member.RoleAttributes))
	if err != nil {
		return diag.Errorf("failed to set role attributes on team member with id %q: %v", member.Id, err)
	}

	return diags
}
