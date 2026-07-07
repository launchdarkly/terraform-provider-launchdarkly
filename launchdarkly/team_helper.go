package launchdarkly

import (
	"fmt"

	ldapi "github.com/launchdarkly/api-client-go/v23"
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
	allRoleKeys, err := fetchAllOffsetPagesWithOptionalInt32Total[string](teamRolesPageLimit, 0, func(offset, limit int64) ([]string, *int32, error) {
		var rolesResponse *ldapi.TeamCustomRoles
		var err error

		err = client.withConcurrency(client.ctx, func() error {
			rolesResponse, _, err = client.ld.TeamsApi.GetTeamRoles(client.ctx, teamKey).
				Limit(limit).
				Offset(offset).
				Execute()
			return err
		})

		if err != nil {
			return nil, nil, fmt.Errorf("failed to get custom roles for team %q: %s", teamKey, handleLdapiErr(err))
		}

		pageRoleKeys := make([]string, 0, len(rolesResponse.Items))
		for _, role := range rolesResponse.Items {
			if role.Key != nil {
				pageRoleKeys = append(pageRoleKeys, *role.Key)
			}
		}

		return pageRoleKeys, rolesResponse.TotalCount, nil
	})
	if err != nil {
		return nil, err
	}

	return allRoleKeys, nil
}

// getAllTeamCustomRoleKeysWithRetry fetches all custom role keys using the 404-retry client.
// This is useful for resources that may experience eventual consistency issues
// (e.g., teams provisioned via Okta team sync).
// Returns an empty slice (not nil) when the team has no roles, which is important for
// Terraform's type system to distinguish between null and empty sets.
func getAllTeamCustomRoleKeysWithRetry(client *Client, teamKey string) ([]string, error) {
	allRoleKeys, err := fetchAllOffsetPagesWithOptionalInt32Total[string](teamRolesPageLimit, 0, func(offset, limit int64) ([]string, *int32, error) {
		var rolesResponse *ldapi.TeamCustomRoles
		var err error

		err = client.withConcurrency(client.ctx, func() error {
			rolesResponse, _, err = client.ld404Retry.TeamsApi.GetTeamRoles(client.ctx, teamKey).
				Limit(limit).
				Offset(offset).
				Execute()
			return err
		})

		if err != nil {
			return nil, nil, fmt.Errorf("failed to get custom roles for team %q: %s", teamKey, handleLdapiErr(err))
		}

		pageRoleKeys := make([]string, 0, len(rolesResponse.Items))
		for _, role := range rolesResponse.Items {
			if role.Key != nil {
				pageRoleKeys = append(pageRoleKeys, *role.Key)
			}
		}

		return pageRoleKeys, rolesResponse.TotalCount, nil
	})
	if err != nil {
		return nil, err
	}

	return allRoleKeys, nil
}

func getAllTeamMembers(client *Client, teamKey string) ([]ldapi.Member, error) {
	allTeamMembers, err := fetchAllOffsetPagesWithOptionalInt32Total[ldapi.Member](teamMemberLimit, 0, func(offset, limit int64) ([]ldapi.Member, *int32, error) {
		var membersResponse *ldapi.Members
		var err error

		err = client.withConcurrency(client.ctx, func() error {
			membersResponse, _, err = client.ld.AccountMembersApi.GetMembers(client.ctx).
				Filter(fmt.Sprintf("team:%s", teamKey)).
				Limit(limit).
				Offset(offset).
				Execute()
			return err
		})

		if err != nil {
			return nil, nil, fmt.Errorf("failed to get members for team %q: %s", teamKey, handleLdapiErr(err))
		}

		return membersResponse.Items, membersResponse.TotalCount, nil
	})
	if err != nil {
		return nil, err
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
	allMaintainers, err := fetchAllOffsetPagesWithOptionalInt32Total[ldapi.MemberSummary](teamMaintainersPageLimit, 0, func(offset, limit int64) ([]ldapi.MemberSummary, *int32, error) {
		var maintainersResponse *ldapi.TeamMaintainers
		var err error

		err = client.withConcurrency(client.ctx, func() error {
			maintainersResponse, _, err = client.ld.TeamsApi.GetTeamMaintainers(client.ctx, teamKey).
				Limit(limit).
				Offset(offset).
				Execute()
			return err
		})

		if err != nil {
			return nil, nil, fmt.Errorf("failed to get maintainers for team %q: %s", teamKey, handleLdapiErr(err))
		}

		return maintainersResponse.Items, maintainersResponse.TotalCount, nil
	})
	if err != nil {
		return nil, err
	}

	return allMaintainers, nil
}
