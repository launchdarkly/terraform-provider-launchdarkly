package launchdarkly

import (
	"fmt"

	ldapi "github.com/launchdarkly/api-client-go/v22"
)

const (
	// teamRolesPageLimit is the number of roles to fetch per API request
	// The API default is 25, but supports higher limits
	teamRolesPageLimit = int64(100)

	// teamMaintainersPageLimit is the number of maintainers to fetch per API request
	// The expand=maintainers parameter has the same 25 item limit as roles
	teamMaintainersPageLimit = int64(100)

	// teamMemberLimit is the number of members to fetch per API request
	// The API max is 1000
	teamMemberLimit = int64(1000)
)

// makeAddAndRemoveArrays returns the set difference (old\new, new\old).
// Used by SDKv2 resources that still reference team schema helpers.
func makeAddAndRemoveArrays(old, updated []string) (remove, add []string) {
	intersection := make(map[string]bool, len(old))
	oldSet := make(map[string]bool, len(old))
	for _, item := range old {
		oldSet[item] = true
	}
	for _, item := range updated {
		if oldSet[item] {
			intersection[item] = true
		}
	}
	for _, item := range old {
		if !intersection[item] {
			remove = append(remove, item)
		}
	}
	for _, item := range updated {
		if !intersection[item] {
			add = append(add, item)
		}
	}
	return remove, add
}

// getAllTeamCustomRoleKeys fetches all custom role keys for a team using pagination.
// The LaunchDarkly API returns a maximum of 25 roles by default when using the expand=roles
// parameter on GetTeam.
// Returns an empty slice (not nil) when the team has no roles, which is important for
// Terraform's type system to distinguish between null and empty sets.
func getAllTeamCustomRoleKeys(client *Client, teamKey string) ([]string, error) {
	allRoleKeys := []string{}
	offset := int64(0)

	for {
		var rolesResponse *ldapi.TeamCustomRoles
		var err error

		err = client.withConcurrency(client.ctx, func() error {
			rolesResponse, _, err = client.ld.TeamsApi.GetTeamRoles(client.ctx, teamKey).
				Limit(teamRolesPageLimit).
				Offset(offset).
				Execute()
			return err
		})

		if err != nil {
			return nil, fmt.Errorf("failed to get custom roles for team %q: %s", teamKey, handleLdapiErr(err))
		}

		// Extract role keys from this page
		for _, role := range rolesResponse.Items {
			if role.Key != nil {
				allRoleKeys = append(allRoleKeys, *role.Key)
			}
		}

		// Check if we've fetched all roles
		totalCount := int64(0)
		if rolesResponse.TotalCount != nil {
			totalCount = int64(*rolesResponse.TotalCount)
		}

		if int64(len(allRoleKeys)) >= totalCount {
			break
		}

		offset += teamRolesPageLimit
	}

	return allRoleKeys, nil
}

// getAllTeamCustomRoleKeysWithRetry fetches all custom role keys using the 404-retry client.
// This is useful for resources that may experience eventual consistency issues
// (e.g., teams provisioned via Okta team sync).
// Returns an empty slice (not nil) when the team has no roles, which is important for
// Terraform's type system to distinguish between null and empty sets.
func getAllTeamCustomRoleKeysWithRetry(client *Client, teamKey string) ([]string, error) {
	allRoleKeys := []string{}
	offset := int64(0)

	for {
		var rolesResponse *ldapi.TeamCustomRoles
		var err error

		err = client.withConcurrency(client.ctx, func() error {
			rolesResponse, _, err = client.ld404Retry.TeamsApi.GetTeamRoles(client.ctx, teamKey).
				Limit(teamRolesPageLimit).
				Offset(offset).
				Execute()
			return err
		})

		if err != nil {
			return nil, fmt.Errorf("failed to get custom roles for team %q: %s", teamKey, handleLdapiErr(err))
		}

		// Extract role keys from this page
		for _, role := range rolesResponse.Items {
			if role.Key != nil {
				allRoleKeys = append(allRoleKeys, *role.Key)
			}
		}

		// Check if we've fetched all roles
		totalCount := int64(0)
		if rolesResponse.TotalCount != nil {
			totalCount = int64(*rolesResponse.TotalCount)
		}

		if int64(len(allRoleKeys)) >= totalCount {
			break
		}

		offset += teamRolesPageLimit
	}

	return allRoleKeys, nil
}

func getAllTeamMembers(client *Client, teamKey string) ([]ldapi.Member, error) {
	allTeamMembers := []ldapi.Member{}
	offset := int64(0)

	for {
		var membersResponse *ldapi.Members
		var err error

		err = client.withConcurrency(client.ctx, func() error {
			membersResponse, _, err = client.ld.AccountMembersApi.GetMembers(client.ctx).
				Filter(fmt.Sprintf("team:%s", teamKey)).
				Limit(teamMemberLimit).
				Offset(offset).
				Execute()
			return err
		})

		if err != nil {
			return nil, fmt.Errorf("failed to get members for team %q: %s", teamKey, handleLdapiErr(err))
		}

		// Append maintainers from this page
		allTeamMembers = append(allTeamMembers, membersResponse.Items...)

		// Check if we've fetched all members
		totalCount := int64(0)
		if membersResponse.TotalCount != nil {
			totalCount = int64(*membersResponse.TotalCount)
		}

		if int64(len(allTeamMembers)) >= totalCount {
			break
		}

		offset += teamMaintainersPageLimit
	}

	return allTeamMembers, nil

}

// getAllTeamMaintainers fetches all maintainers for a team using pagination.
// The LaunchDarkly API returns a maximum of 25 maintainers by default when using the expand=maintainers
// parameter on GetTeam. For teams with more than 25 maintainers, we need to use the dedicated
// GetTeamMaintainers endpoint with pagination.
// Returns an empty slice (not nil) when the team has no maintainers, which is important for
// Terraform's type system to distinguish between null and empty sets.
// See: https://launchdarkly.atlassian.net/browse/REL-11737
func getAllTeamMaintainers(client *Client, teamKey string) ([]ldapi.MemberSummary, error) {
	allMaintainers := []ldapi.MemberSummary{}
	offset := int64(0)

	for {
		var maintainersResponse *ldapi.TeamMaintainers
		var err error

		err = client.withConcurrency(client.ctx, func() error {
			maintainersResponse, _, err = client.ld.TeamsApi.GetTeamMaintainers(client.ctx, teamKey).
				Limit(teamMaintainersPageLimit).
				Offset(offset).
				Execute()
			return err
		})

		if err != nil {
			return nil, fmt.Errorf("failed to get maintainers for team %q: %s", teamKey, handleLdapiErr(err))
		}

		// Append maintainers from this page
		allMaintainers = append(allMaintainers, maintainersResponse.Items...)

		// Check if we've fetched all maintainers
		totalCount := int64(0)
		if maintainersResponse.TotalCount != nil {
			totalCount = int64(*maintainersResponse.TotalCount)
		}

		if int64(len(allMaintainers)) >= totalCount {
			break
		}

		offset += teamMaintainersPageLimit
	}

	return allMaintainers, nil
}
