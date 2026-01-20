package launchdarkly

import (
	"fmt"

	ldapi "github.com/launchdarkly/api-client-go/v17"
)

const (
	// teamRolesPageLimit is the number of roles to fetch per API request
	// The API default is 20, but supports higher limits
	teamRolesPageLimit = int64(100)
)

// getAllTeamCustomRoleKeys fetches all custom role keys for a team using pagination.
// The LaunchDarkly API returns a maximum of 20 roles by default when using the expand=roles
// parameter on GetTeam. For teams with more than 20 roles, we need to use the dedicated
// GetTeamRoles endpoint with pagination.
// See: https://launchdarkly.atlassian.net/browse/REL-11737
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
