package launchdarkly

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"

	ldapi "github.com/launchdarkly/api-client-go/v23"
)

// getRandomSleepDuration returns a duration between [0, maxDuration)
func getRandomSleepDuration(maxDuration time.Duration) time.Duration {
	n := rand.Int63n(int64(maxDuration))
	return time.Duration(n)
}

func intPtr(i int) *int {
	return &i
}

func strPtr(v string) *string { return &v }

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

// isApprovalRequiredErr reports whether err is a LaunchDarkly API rejection
// caused by an approval requirement. When approvals are enabled for a
// resource, the API rejects direct mutations with HTTP 403 and the body
// {"code":"forbidden","message":"approval is required"}. Segment approvals
// gate targeting changes (included / excluded / rules / *_contexts) and,
// unlike flag approvals, cannot yet be bypassed by a service token
// (FROPS-190) — so Terraform cannot apply gated segment changes. See issue
// #370. Keyed on the message rather than the bare 403 so ordinary
// permission-denied 403s are not misclassified.
func isApprovalRequiredErr(err error) bool {
	if err == nil {
		return false
	}
	const marker = "approval is required"
	// The API client returns the response payload via Body(); err.Error() only
	// carries the HTTP status line, so check the body first.
	if swaggerErr, ok := err.(*ldapi.GenericOpenAPIError); ok {
		if strings.Contains(string(swaggerErr.Body()), marker) {
			return true
		}
	}
	return strings.Contains(err.Error(), marker)
}

func isStatusNotFound(response *http.Response) bool {
	if response != nil && response.StatusCode == http.StatusNotFound {
		return true
	}
	return false
}

func isStatusConflict(response *http.Response) bool {
	if response != nil && response.StatusCode == http.StatusConflict {
		return true
	}
	return false
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

func addForceNewDescription(description string, forceNew bool) string {
	if forceNew {
		description += " A change in this field forces the destruction of the existing resource and the creation of a new one."
	}
	return description
}

func oxfordCommaJoin(str []string) string {
	output := ""
	for idx, key := range str {
		output += fmt.Sprintf("`%s`", key)
		if idx < len(str)-2 {
			output += ", "
		} else if idx == len(str)-2 {
			output += ", and "
		}
	}
	return output
}

func splitID(id string, expectedParts int) []string {
	parts := strings.Split(id, "/")
	if len(parts) != expectedParts {
		return nil
	}
	return parts
}
