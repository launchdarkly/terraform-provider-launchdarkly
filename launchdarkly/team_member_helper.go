package launchdarkly

import (
	"fmt"
	"net/http"

	ldapi "github.com/launchdarkly/api-client-go/v17"
)

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
