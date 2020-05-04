package launchdarkly

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	ldapi "github.com/launchdarkly/api-client-go"
)

const (
	MAX_409_RETRIES = 5
	MAX_429_RETRIES = 5
)

func handleRateLimit(apiCall func() (interface{}, *http.Response, error)) (interface{}, *http.Response, error) {
	obj, res, err := apiCall()
	for retryCount := 0; res != nil && res.StatusCode == http.StatusTooManyRequests && retryCount < MAX_429_RETRIES; retryCount++ {
		log.Println("[DEBUG] received a 429 Too Many Requests error. retrying")
		resetStr := res.Header.Get("X-RateLimit-Reset")
		resetInt, parseErr := strconv.ParseInt(resetStr, 10, 64)
		if parseErr != nil {
			randomRetrySleep()
		} else {
			resetTime := time.Unix(0, resetInt*int64(time.Millisecond))
			sleepDuration := time.Until(resetTime)
			log.Println("[DEBUG] sleeping", sleepDuration)
			time.Sleep(sleepDuration)
		}
		obj, res, err = apiCall()
	}
	return obj, res, err

}

func handleNoConflict(apiCall func() (interface{}, *http.Response, error)) (interface{}, *http.Response, error) {
	obj, res, err := apiCall()
	for retryCount := 0; res != nil && res.StatusCode == http.StatusConflict && retryCount < MAX_409_RETRIES; retryCount++ {
		log.Println("[DEBUG] received a 409 conflict. retrying")
		randomRetrySleep()
		obj, res, err = apiCall()
	}
	return obj, res, err
}

var randomRetrySleepSeeded = false

// Sleep for a random interval between 200ms and 500ms
func randomRetrySleep() {
	if !randomRetrySleepSeeded {
		rand.Seed(time.Now().UnixNano())
	}
	n := rand.Intn(300) + 200
	time.Sleep(time.Duration(n) * time.Millisecond)
}

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

func stringSliceToInterfaceSlice(input []string) []interface{} {
	o := make([]interface{}, 0, len(input))
	for _, v := range input {
		o = append(o, v)
	}
	return o
}
