package launchdarkly

import (
	"fmt"
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

// handleLdapiErr extracts the error message and body from a ldapi.GenericSwaggerError or simply returns the
// error message if it is not a ldapi.GenericSwaggerError
func handleLdapiErr(err error) string {
	if err == nil {
		return ""
	}
	if swaggerErr, ok := err.(ldapi.GenericSwaggerError); ok {
		return fmt.Sprintf("%s: %s", swaggerErr.Error(), string(swaggerErr.Body()))
	}

	return err.Error()

}
