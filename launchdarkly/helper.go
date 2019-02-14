package launchdarkly

import (
	ldapi "github.com/launchdarkly/api-client-go"
)

func ptr(v interface{}) *interface{} { return &v }
func stringPtr(v string) *string     { return &v }

func stringList(v []interface{}) []string {
	list := make([]string, len(v))
	for i, elem := range v {
		list[i] = elem.(string)
	}
	return list
}

func patchReplace(path string, value interface{}) ldapi.PatchOperation {
	return ldapi.PatchOperation{
		Op:    "replace",
		Path:  path,
		Value: &value,
	}
}
