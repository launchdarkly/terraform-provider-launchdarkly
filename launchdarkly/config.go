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

	"golang.org/x/sync/semaphore"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	ldapi "github.com/launchdarkly/api-client-go/v17"
)

//nolint:staticcheck // The version string gets updated at build time using -ldflags
var version = "unreleased"

const (
	APIVersion              = "20240415"
	MAX_RETRIES             = 12
	RETRY_WAIT_MIN          = 200 * time.Millisecond
	RETRY_WAIT_MAX          = 10000 * time.Millisecond
	DEFAULT_MAX_CONCURRENCY = 1 // provider-specific parallelism to avoid hitting rate limits / timeouts
)

// Client is used by the provider to access the ld API.
type Client struct {
	apiKey  string
	apiHost string

	// ld is the standard API client that we use in most cases to interact with LaunchDarkly's APIs.
	ld *ldapi.APIClient

	ldBeta *ldapi.APIClient

	// ld404Retry is the same as ld except that it will also retry 404s with an exponential backoff. In most cases `ld` should be used instead. sc-218015
	ld404Retry     *ldapi.APIClient
	ctx            context.Context
	fallbackClient *http.Client

	semaphore *semaphore.Weighted
}

func (c *Client) withConcurrency(ctx context.Context, fn func() error) error {
	// this will block if the semaphore has reached its maxConcurrent requests
	if err := c.semaphore.Acquire(ctx, 1); err != nil {
		return fmt.Errorf("failed to acquire semaphore: %w", err)
	}
	defer c.semaphore.Release(1)

	return fn()
}

// we pass maxConcurrent through here so that we can set it differently for tests
func newClient(token string, apiHost string, oauth bool, httpTimeoutSeconds, maxConcurrent int) (*Client, error) {
	return baseNewClient(token, apiHost, oauth, httpTimeoutSeconds, APIVersion, maxConcurrent)
}

func newBetaClient(token string, apiHost string, oauth bool, httpTimeoutSeconds, maxConcurrent int) (*Client, error) {
	return baseNewClient(token, apiHost, oauth, httpTimeoutSeconds, "beta", maxConcurrent)
}

func newLDClientConfig(apiHost string, httpTimeoutSeconds int, apiVersion string, retryPolicy retryablehttp.CheckRetry) *ldapi.Configuration {
	cfg := ldapi.NewConfiguration()
	cfg.Host = apiHost
	cfg.DefaultHeader = make(map[string]string)
	cfg.UserAgent = fmt.Sprintf("launchdarkly-terraform-provider/%s", version)
	cfg.HTTPClient = newRetryableClient(retryPolicy)
	cfg.HTTPClient.Timeout = time.Duration(httpTimeoutSeconds) * time.Second
	cfg.AddDefaultHeader("LD-API-Version", apiVersion)
	return cfg
}

func baseNewClient(token string, apiHost string, oauth bool, httpTimeoutSeconds int, apiVersion string, maxConcurrent int) (*Client, error) {
	if token == "" {
		return nil, errors.New("token cannot be empty")
	}

	standardConfig := newLDClientConfig(apiHost, httpTimeoutSeconds, apiVersion, standardRetryPolicy)
	betaConfig := newLDClientConfig(apiHost, httpTimeoutSeconds, "beta", standardRetryPolicy)
	configWith404Retries := newLDClientConfig(apiHost, httpTimeoutSeconds, apiVersion, retryPolicyWith404Retries)

	ctx := context.WithValue(context.Background(), ldapi.ContextAPIKeys, map[string]ldapi.APIKey{
		"ApiKey": {
			Key: token,
		}})
	if oauth {
		ctx = context.WithValue(context.Background(), ldapi.ContextAccessToken, token)
	}

	// TODO: remove this once we get the go client reset endpoint fixed
	fallbackClient := newRetryableClient(standardRetryPolicy)
	fallbackClient.Timeout = time.Duration(5 * time.Second)

	return &Client{
		apiKey:         token,
		apiHost:        apiHost,
		ld:             ldapi.NewAPIClient(standardConfig),
		ldBeta:         ldapi.NewAPIClient(betaConfig),
		ld404Retry:     ldapi.NewAPIClient(configWith404Retries),
		ctx:            ctx,
		fallbackClient: fallbackClient,
		semaphore:      semaphore.NewWeighted(int64(maxConcurrent)),
	}, nil
}

func newRetryableClient(retryPolicy retryablehttp.CheckRetry) *http.Client {
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

func standardRetryPolicy(ctx context.Context, resp *http.Response, err error) (bool, error) {
	return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
}

// retryPolicyWith404Retries extends our standard retryPolicy but also retries 404s (with exponential backoff).
// This should be used sparingly as 404 typically denote the resource has been deleted. sc-218015
func retryPolicyWith404Retries(ctx context.Context, resp *http.Response, err error) (bool, error) {
	retry, retryErr := standardRetryPolicy(ctx, resp, err)
	if !retry && retryErr == nil && err == nil && resp.StatusCode == http.StatusNotFound {
		log.Println("[DEBUG] received a 404 from LaunchDarkly. Retrying.")
		return true, nil
	}

	return retry, retryErr
}
