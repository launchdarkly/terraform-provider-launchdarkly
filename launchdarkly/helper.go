package launchdarkly

import (
	"fmt"
	"net/http"

	ldapi "github.com/launchdarkly/api-client-go"
)

func ptr(v interface{}) *interface{} { return &v }

func intPtr(i int) *int {
	return &i
}

func patchReplace(path string, value interface{}) ldapi.PatchOperation {
	return ldapi.PatchOperation{
		Op:    "replace",
		Path:  path,
		Value: &value,
	}
}

func patchAdd(path string, value interface{}) ldapi.PatchOperation {
	return ldapi.PatchOperation{
		Op:    "add",
		Path:  path,
		Value: &value,
	}
}

func patchRemove(path string) ldapi.PatchOperation {
	return ldapi.PatchOperation{
		Op:   "remove",
		Path: path,
	}
}

// handleLdapiErr extracts the error message and body from a ldapi.GenericSwaggerError or simply returns the
// error  if it is not a ldapi.GenericSwaggerError
func handleLdapiErr(err error) error {
	if err == nil {
		return nil
	}
	if swaggerErr, ok := err.(ldapi.GenericSwaggerError); ok {
		return fmt.Errorf("%s: %s", swaggerErr.Error(), string(swaggerErr.Body()))
	}
	return err
}

func isStatusNotFound(response *http.Response) bool {
	if response != nil && response.StatusCode == http.StatusNotFound {
		return true
	}
	return false
}
