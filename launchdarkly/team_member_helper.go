package launchdarkly

import (
	"fmt"
)

// The LD api returns custom role IDs (not keys). Since we want to set custom_roles with keys, we need to look up their IDs
func customRoleIDsToKeys(client *Client, ids []string) ([]string, error) {
	customRoleKeys := make([]string, 0, len(ids))
	for _, customRoleID := range ids {
		role, res, err := client.ld.CustomRolesApi.GetCustomRole(client.ctx, customRoleID).Execute()
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
		role, res, err := client.ld.CustomRolesApi.GetCustomRole(client.ctx, key).Execute()
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
