package launchdarkly

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v12"
)

var randomRetrySleepSeeded = false

// getRandomSleepDuration returns a duration between [0, maxDuration)
func getRandomSleepDuration(maxDuration time.Duration) time.Duration {
	if !randomRetrySleepSeeded {
		rand.Seed(time.Now().UnixNano())
	}
	n := rand.Int63n(int64(maxDuration))
	return time.Duration(n)
}

func ptr(v interface{}) *interface{} { return &v }

func intPtr(i int) *int {
	return &i
}

func strPtr(v string) *string { return &v }

func strArrayPtr(v []string) *[]string { return &v }

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
	if swaggerErr, ok := err.(*ldapi.GenericOpenAPIError); ok {
		return fmt.Errorf("%s: %s", swaggerErr.Error(), string(swaggerErr.Body()))
	}
	return err
}

func isTimeoutError(err error) bool {
	e, ok := err.(net.Error)
	return ok && e.Timeout()
}

func isStatusNotFound(response *http.Response) bool {
	if response != nil && response.StatusCode == http.StatusNotFound {
		return true
	}
	return false
}

func stringSliceToInterfaceSlice(input []string) []interface{} {
	o := make([]interface{}, 0, len(input))
	for _, v := range input {
		o = append(o, v)
	}
	return o
}

func interfaceSliceToStringSlice(input []interface{}) []string {
	o := make([]string, 0, len(input))
	for _, v := range input {
		o = append(o, v.(string))
	}
	return o
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// removeInvalidFieldsForDataSource removes all default and validation functions from the schema map.
// This is done because Terraform requires defaults and validation functions to be nil for read-only data-source attributes.
func removeInvalidFieldsForDataSource(schemaMap map[string]*schema.Schema) map[string]*schema.Schema {
	for k, v := range schemaMap {
		if v.Computed {
			v.Default = nil
			v.ValidateDiagFunc = nil
			v.DiffSuppressFunc = nil
			v.MaxItems = 0
		}
		schemaMap[k] = v
	}
	return schemaMap
}
