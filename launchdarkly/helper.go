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
	MAX_429_RETRIES = 20
)

func handleRateLimit(apiCall func() (interface{}, *http.Response, error)) (interface{}, *http.Response, error) {
	obj, res, err := apiCall()
	for retryCount := 0; res != nil && res.StatusCode == http.StatusTooManyRequests && retryCount < MAX_429_RETRIES; retryCount++ {
		log.Println("[DEBUG] received a 429 Too Many Requests error. retrying")
		resetStr := res.Header.Get("X-RateLimit-Reset")
		resetInt, parseErr := strconv.ParseInt(resetStr, 10, 64)
		if parseErr != nil {
			log.Println("[DEBUG] could not parse X-RateLimit-Reset header. Sleeping for a random interval.")
			randomRetrySleep()
		} else {
			resetTime := time.Unix(0, resetInt*int64(time.Millisecond))
			sleepDuration := time.Until(resetTime)

			// We have observed situations where LD-s retry header results in a negative sleep duration. In this case,
			// multiply the duration by -1 and add jitter
			if sleepDuration <= 0 {
				log.Printf("[DEBUG] received a negative rate limit retry duration of %s.", sleepDuration)
				sleepDuration = -1 * sleepDuration
			}
			sleepDurationWithJitter := sleepDuration + getRandomSleepDuration(sleepDuration)
			log.Println("[DEBUG] sleeping", sleepDurationWithJitter)
			time.Sleep(sleepDurationWithJitter)
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

// getRandomSleepDuration returns a duration between [0, maxDuration)
func getRandomSleepDuration(maxDuration time.Duration) time.Duration {
	if !randomRetrySleepSeeded {
		rand.Seed(time.Now().UnixNano())
	}
	n := rand.Int63n(int64(maxDuration))
	return time.Duration(n)
}

// Sleep for a random interval between 200ms and 500ms
func randomRetrySleep() {
	duration := 200*time.Millisecond + getRandomSleepDuration(300*time.Millisecond)
	time.Sleep(duration)
}

func ptr(v interface{}) *interface{} { return &v }

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
