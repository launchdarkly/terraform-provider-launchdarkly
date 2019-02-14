package launchdarkly

import (
	ldapi "github.com/launchdarkly/api-client-go"
)

func ptr(v interface{}) *interface{} { return &v }

func patchReplace(path string, value interface{}) ldapi.PatchOperation {
	return ldapi.PatchOperation{
		Op:    "replace",
		Path:  path,
		Value: &value,
	}
}
