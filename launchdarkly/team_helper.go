package launchdarkly

import (
	"fmt"

	ldapi "github.com/launchdarkly/api-client-go/v17"
)

const (
	// teamRolesPageLimit is the number of roles to fetch per API request
	// The API default is 25, but supports higher limits
	teamRolesPageLimit = int64(100)

	// teamMaintainersPageLimit is the number of maintainers to fetch per API request
	// The expand=maintainers parameter has the same 25 item limit as roles
	teamMaintainersPageLimit = int64(100)
)

// getAllTeamCustomRoleKeys fetches all custom role keys for a team using pagination.
// The LaunchDarkly API returns a maximum of 25 roles by default when using the expand=roles
// parameter on GetTeam.
func getAllTeamCustomRoleKeys(client *Client, teamKey string) ([]string, error) {
	var allRoleKeys []string
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
func getAllTeamCustomRoleKeysWithRetry(client *Client, teamKey string) ([]string, error) {
	var allRoleKeys []string
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

// getAllTeamMaintainers fetches all maintainers for a team using pagination.
// The LaunchDarkly API returns a maximum of 25 maintainers by default when using the expand=maintainers
// parameter on GetTeam. For teams with more than 25 maintainers, we need to use the dedicated
// GetTeamMaintainers endpoint with pagination.
// See: https://launchdarkly.atlassian.net/browse/REL-11737
func getAllTeamMaintainers(client *Client, teamKey string) ([]ldapi.MemberSummary, error) {
	var allMaintainers []ldapi.MemberSummary
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
