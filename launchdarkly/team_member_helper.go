package launchdarkly

import (
	"fmt"
	"net/http"
	"strings"

	ldapi "github.com/launchdarkly/api-client-go/v22"
)

// getTeamMemberByEmail performs a paginated GetMembers search filtered
// by email and returns the first exact match. Shared by the team_member
// and team_members data sources.
func getTeamMemberByEmail(client *Client, memberEmail string) (*ldapi.Member, error) {
	emailFilter := fmt.Sprintf("email:%s", memberEmail)
	expand := "roleAttributes"
	members, err := getMembersPaginated(client, &emailFilter, &expand, nil, 1, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get team member %q: %v", memberEmail, handleLdapiErr(err))
	}
	if len(members) < 1 {
		return nil, fmt.Errorf("no member found with email %q", memberEmail)
	}
	return &members[0], nil
}

func getTeamMembersByEmail(client *Client, memberEmails []string) ([]ldapi.Member, error) {
	if len(memberEmails) == 0 {
		return []ldapi.Member{}, nil
	}
	emailFilter := fmt.Sprintf("email:%s", strings.Join(memberEmails, "|"))
	expand := "roleAttributes"
	members, err := getMembersPaginated(client, &emailFilter, &expand, nil, teamMemberLimit, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get team members by emails: %v", handleLdapiErr(err))
	}
	return members, nil
}

func getMembersPaginated(client *Client, filter, expand, sort *string, limit int64, initialOffset *int64) ([]ldapi.Member, error) {
	offset := int64(0)
	if initialOffset != nil {
		offset = *initialOffset
	}

	memberItems, err := fetchAllOffsetPagesWithOptionalInt32Total[ldapi.Member](limit, offset, func(offset, limit int64) ([]ldapi.Member, *int32, error) {
		var members *ldapi.Members
		var err error
		err = client.withConcurrency(client.ctx, func() error {
			request := client.ld.AccountMembersApi.GetMembers(client.ctx).Offset(offset).Limit(limit)
			if filter != nil {
				request = request.Filter(*filter)
			}
			if expand != nil {
				request = request.Expand(*expand)
			}
			if sort != nil {
				request = request.Sort(*sort)
			}
			members, _, err = request.Execute()
			return err
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get team members by emails: %v", handleLdapiErr(err))
		}
		return members.Items, members.TotalCount, nil
	})
	if err != nil {
		return nil, err
	}

	return memberItems, nil
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
