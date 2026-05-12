package launchdarkly

import (
	"fmt"
	"net/http"
	"net/url"

	ldapi "github.com/launchdarkly/api-client-go/v22"
)

// getTeamMemberByEmail performs a paginated GetMembers search filtered
// by email and returns the first exact match. Lives here (not in the
// SDKv2 data_source_launchdarkly_team_member.go that owned it pre-
// migration) because both the framework team_member + team_members
// data sources consume it after the data source was migrated to
// terraform-plugin-framework in Phase 1.3.2.
func getTeamMemberByEmail(client *Client, memberEmail string) (*ldapi.Member, error) {
	teamMemberLimit := int64(1000)

	var members *ldapi.Members
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		members, _, err = client.ld.AccountMembersApi.GetMembers(client.ctx).Filter(fmt.Sprintf("query:%s", url.QueryEscape(memberEmail))).Expand("roleAttributes").Execute()
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read team member with email: %s: %v", memberEmail, handleLdapiErr(err))
	}

	totalMemberCount := int(*members.TotalCount)
	memberItems := members.Items
	membersPulled := len(memberItems)
	for membersPulled < totalMemberCount {
		offset := int64(membersPulled)
		var newMembers *ldapi.Members
		err = client.withConcurrency(client.ctx, func() error {
			newMembers, _, err = client.ld.AccountMembersApi.GetMembers(client.ctx).Limit(teamMemberLimit).Offset(offset).Filter(fmt.Sprintf("query:%s", url.QueryEscape(memberEmail))).Execute()
			return err
		})
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

// The LD api returns custom role IDs (not keys). Since we want to set custom_roles with keys, we need to look up their IDs
func customRoleIDsToKeys(client *Client, ids []string) ([]string, error) {
	customRoleKeys := make([]string, 0, len(ids))
	for _, customRoleID := range ids {
		var role *ldapi.CustomRole
		var res *http.Response
		var err error
		err = client.withConcurrency(client.ctx, func() error {
			role, res, err = client.ld.CustomRolesApi.GetCustomRole(client.ctx, customRoleID).Execute()
			return err
		})
		if isStatusNotFound(res) {
			return nil, fmt.Errorf("failed to find custom role key for ID %q", customRoleID)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve custom role key for role ID %q: %v", customRoleID, err)
		}
		customRoleKeys = append(customRoleKeys, role.Key)
	}
	return customRoleKeys, nil
}

// Since the LD API expects custom role IDs, we need to look up each key to retrieve the ID
func customRoleKeysToIDs(client *Client, keys []string) ([]string, error) {
	customRoleIds := make([]string, 0, len(keys))
	for _, key := range keys {
		var role *ldapi.CustomRole
		var res *http.Response
		var err error
		err = client.withConcurrency(client.ctx, func() error {
			role, res, err = client.ld.CustomRolesApi.GetCustomRole(client.ctx, key).Execute()
			return err
		})
		if isStatusNotFound(res) {
			return nil, fmt.Errorf("failed to find custom ID for key %q", key)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve custom role ID for key %q: %v", key, err)
		}
		customRoleIds = append(customRoleIds, role.Id)
	}
	return customRoleIds, nil
}
