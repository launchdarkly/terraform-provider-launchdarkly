package launchdarkly

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v17"
)

func dataSourceTeamMembers() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceTeamMembersRead,
		Schema: map[string]*schema.Schema{
			EMAILS: {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "An array of unique email addresses associated with the team members.",
			},
			IGNORE_MISSING: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "A boolean to determine whether to ignore members that weren't found.",
			},
			TEAM_MEMBERS: {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: memberSchema(),
				},
				Description: "The members that were found. The following attributes are available for each member:\n\n- `id` - The 24 character alphanumeric ID of the team member.\n\n- `first_name` - The team member's given name.\n\n- `last_name` - The team member's family name.\n\n- `role` - The role associated with team member. Possible roles are `owner`, `reader`, `writer`, or `admin`.\n\n- `custom_roles` - (Optional) The list of custom roles keys associated with the team member. Custom roles are only available to customers on an Enterprise plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).\n",
			},
		},
		Description: `Provides a LaunchDarkly team members data source.

This data source allows you to retrieve team member information from your LaunchDarkly organization on multiple team members.`,
	}
}

func dataSourceTeamMembersRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*Client)
	var members []ldapi.Member
	expectedCount := 0
	ignoreMissing := d.Get(IGNORE_MISSING).(bool)

	// Get our members
	// There are tradeoffs to be had here
	// We've decided to get all the members and filter in code for now, in order to not scale the amount of requests with team_member list size
	if emails, ok := d.Get(EMAILS).([]interface{}); ok && len(emails) > 0 {
		expectedCount = len(emails)
		allMembers, err := getAllTeamMembers(client)
		if err != nil {
			return diag.FromErr(err)
		}
		for _, memberEmail := range emails {
			var member ldapi.Member
			memberFound := false
			for _, foundMember := range allMembers {
				if foundMember.Email == memberEmail {
					member = foundMember
					memberFound = true
					break
				}
			}
			if !memberFound {
				if ignoreMissing {
					continue
				}
				return diag.Errorf("No team member found for email: %s", memberEmail)
			}
			members = append(members, member)
		}
	}

	if !ignoreMissing && len(members) != expectedCount {
		return diag.Errorf("unexpected number of users returned (%d != %d)", len(members), expectedCount)
	}

	// Build our member list
	ids := make([]string, 0, len(members))
	memberList := make([]map[string]interface{}, 0, len(members))
	for _, m := range members {
		member := make(map[string]interface{})
		member[ID] = m.Id
		member[EMAIL] = m.Email
		member[FIRST_NAME] = m.FirstName
		member[LAST_NAME] = m.LastName
		member[ROLE] = m.Role
		member[CUSTOM_ROLES] = m.CustomRoles
		memberList = append(memberList, member)
		ids = append(ids, m.Id)
	}

	// Build an ID out of a hash of all the team members ids
	h := sha1.New()
	if _, err := h.Write([]byte(strings.Join(ids, "-"))); err != nil {
		return diag.Errorf("unable to compute hash for IDs: %v", err)
	}
	d.SetId("team_members#" + base64.URLEncoding.EncodeToString(h.Sum(nil)))

	err := d.Set(TEAM_MEMBERS, memberList)

	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func getAllTeamMembers(client *Client) ([]ldapi.Member, error) {
	// this should be the max limit allowed when the member-list-max-limit flag is on
	teamMemberLimit := int64(1000)

	// After changing this to query by member email, we shouldn't need the limit and recursion on requests, but leaving it in just to be extra safe
	var members *ldapi.Members
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		members, _, err = client.ld.AccountMembersApi.GetMembers(client.ctx).Limit(teamMemberLimit).Execute()
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to read team members: %v", handleLdapiErr(err))
	}

	totalMemberCount := int(*members.TotalCount)

	memberItems := members.Items
	membersPulled := len(memberItems)
	for membersPulled < totalMemberCount {
		offset := int64(membersPulled)
		var newMembers *ldapi.Members
		err = client.withConcurrency(client.ctx, func() error {
			newMembers, _, err = client.ld.AccountMembersApi.GetMembers(client.ctx).Limit(teamMemberLimit).Offset(offset).Execute()
			return err
		})

		if err != nil {
			return nil, fmt.Errorf("failed to read team members: %v", handleLdapiErr(err))
		}

		memberItems = append(memberItems, newMembers.Items...)
		membersPulled = len(memberItems)
	}

	return memberItems, nil

}
