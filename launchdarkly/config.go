package launchdarkly

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	ldapi "github.com/launchdarkly/api-client-go/v12"
)

//nolint:staticcheck // The version string gets updated at build time using -ldflags
var version = "unreleased"

const (
	APIVersion     = "20220603"
	MAX_RETRIES    = 8
	RETRY_WAIT_MIN = 200 * time.Millisecond
	RETRY_WAIT_MAX = 2000 * time.Millisecond
)

// Client is used by the provider to access the ld API.
type Client struct {
	apiKey         string
	apiHost        string
	ld             *ldapi.APIClient
	ctx            context.Context
	fallbackClient *http.Client
}

func newClient(token string, apiHost string, oauth bool, httpTimeoutSeconds int) (*Client, error) {
	return baseNewClient(token, apiHost, oauth, httpTimeoutSeconds, APIVersion)
}

func newBetaClient(token string, apiHost string, oauth bool, httpTimeoutSeconds int) (*Client, error) {
	return baseNewClient(token, apiHost, oauth, httpTimeoutSeconds, "beta")
}

func baseNewClient(token string, apiHost string, oauth bool, httpTimeoutSeconds int, apiVersion string) (*Client, error) {
	if token == "" {
		return nil, errors.New("token cannot be empty")
	}

	cfg := ldapi.NewConfiguration()
	cfg.Host = apiHost
	cfg.DefaultHeader = make(map[string]string)
	cfg.UserAgent = fmt.Sprintf("launchdarkly-terraform-provider/%s", version)
	cfg.HTTPClient = newRetryableClient()
	cfg.HTTPClient.Timeout = time.Duration(httpTimeoutSeconds) * time.Second
	cfg.AddDefaultHeader("LD-API-Version", apiVersion)

	ctx := context.WithValue(context.Background(), ldapi.ContextAPIKeys, map[string]ldapi.APIKey{
		"ApiKey": {
			Key: token,
		}})
	if oauth {
		ctx = context.WithValue(context.Background(), ldapi.ContextAccessToken, token)
	}

	// TODO: remove this once we get the go client reset endpoint fixed
	fallbackClient := newRetryableClient()
	fallbackClient.Timeout = time.Duration(5 * time.Second)

	return &Client{
		apiKey:         token,
		apiHost:        apiHost,
		ld:             ldapi.NewAPIClient(cfg),
		ctx:            ctx,
		fallbackClient: fallbackClient,
	}, nil
}

func newRetryableClient() *http.Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryWaitMin = RETRY_WAIT_MIN
	retryClient.RetryWaitMax = RETRY_WAIT_MAX
	retryClient.Backoff = backOff
	retryClient.CheckRetry = retryPolicy
	retryClient.RetryMax = MAX_RETRIES
	retryClient.ErrorHandler = retryablehttp.PassthroughErrorHandler

	return retryClient.StandardClient()
}

func backOff(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
	if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
		sleepStr := resp.Header.Get("X-RateLimit-Reset")
		if sleep, err := strconv.ParseInt(sleepStr, 10, 64); err == nil {
			resetTime := time.Unix(0, sleep*int64(time.Millisecond))
			sleepDuration := time.Until(resetTime)

			// We have observed situations where LD-s retry header results in a negative sleep duration. In this case,
			// multiply the duration by -1 and add jitter
			if sleepDuration <= 0 {
				log.Printf("[DEBUG] received a negative rate limit retry duration of %s.", sleepDuration)
				sleepDuration = -1 * sleepDuration
			}

			return sleepDuration + getRandomSleepDuration(sleepDuration)
		}
	}

	backoffTime := math.Pow(2, float64(attemptNum)) * float64(min)
	sleep := time.Duration(backoffTime)
	if float64(sleep) != backoffTime || sleep > max {
		sleep = max
	}
	return sleep
}

func retryPolicy(ctx context.Context, resp *http.Response, err error) (bool, error) {
	retry, retryErr := retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	if !retry && retryErr == nil && err == nil && resp.StatusCode == http.StatusConflict {
		return true, nil
	}

	return retry, retryErr
}
