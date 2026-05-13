package launchdarkly

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

// LD's IP allowlist API persists the whole allowlist as a single account
// document. Concurrent writes (POST/PATCH/DELETE on /api/v2/account/ip-allowlist)
// race on the document's underlying version and the loser comes back with a
// 409 "Conflict creating IP allowlist entry" even when the IPs themselves
// don't collide. The acceptance tests in this file each call resource.ParallelTest
// so they fan out within a single shard, and concurrent CI runs against the
// same blitz LD account add more contention. The retryablehttp policy that
// powers fallbackClient does not retry 409s, so a single lost race is enough
// to fail the build.
//
// Belt-and-suspenders mitigation:
//
//  1. ipAllowlistWriteMu serialises POST/PATCH/DELETE so two goroutines
//     inside one test binary (e.g. resource.ParallelTest steps that
//     overlap) take turns. This eliminates the dominant in-process race
//     that CI's terraform-plugin-testing matrix surfaced.
//  2. ipAllowlistRequest retries 409 with jittered backoff so any
//     remaining cross-process contention (concurrent CI runs against the
//     same account, etc.) still converges.
//
// Reads (GET) skip the mutex; 409 retry stays uniform for the rare case
// the API surfaces a transient read conflict.
var ipAllowlistWriteMu sync.Mutex

const (
	ipAllowlistMaxAttempts = 4
	ipAllowlistBaseBackoff = 200 * time.Millisecond
)

func ipAllowlistBackoff(attempt int) time.Duration {
	d := ipAllowlistBaseBackoff * time.Duration(1<<attempt) //nolint:gosec // attempt is bounded by ipAllowlistMaxAttempts
	jitter := time.Duration(rand.Int63n(int64(ipAllowlistBaseBackoff)))
	return d + jitter
}

func ipAllowlistMethodMutates(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete:
		return true
	}
	return false
}

type ipAllowlistResponse struct {
	SessionAllowlistEnabled  bool                       `json:"sessionAllowlistEnabled"`
	ApiTokenAllowlistEnabled bool                       `json:"apiTokenAllowlistEnabled"`
	Entries                  []ipAllowlistEntryResponse `json:"entries"`
}

type ipAllowlistEntryResponse struct {
	ID          string  `json:"_id"`
	IpAddress   string  `json:"ipAddress"`
	Description *string `json:"description,omitempty"`
}

type createIpAllowlistEntryRequest struct {
	IpAddress   string  `json:"ipAddress"`
	Description *string `json:"description,omitempty"`
}

type patchIpAllowlistEntryRequest struct {
	Description string `json:"description"`
}

type patchIpAllowlistConfigRequest struct {
	SessionAllowlistEnabled  *bool `json:"sessionAllowlistEnabled,omitempty"`
	ApiTokenAllowlistEnabled *bool `json:"apiTokenAllowlistEnabled,omitempty"`
}

type ipAllowlistErrorResponse struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

const ipAllowlistBasePath = "/api/v2/account/ip-allowlist"

func ipAllowlistRequest(client *Client, method, path string, body interface{}) (*http.Response, []byte, error) {
	// Marshal once; the request body itself is short and re-using the
	// buffer per attempt is simpler than serializing on every retry.
	var rawBody []byte
	if body != nil {
		var err error
		rawBody, err = json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	endpoint := fmt.Sprintf("%s%s", client.apiHost, path)
	if !strings.HasPrefix(endpoint, "http") {
		endpoint = "https://" + endpoint
	}

	// Mutating calls serialise in-process so two goroutines inside the
	// same test binary don't race the LD account allowlist version.
	// Cross-process contention falls back to the 409 retry loop below.
	if ipAllowlistMethodMutates(method) {
		ipAllowlistWriteMu.Lock()
		defer ipAllowlistWriteMu.Unlock()
	}

	var (
		resp     *http.Response
		respBody []byte
		lastErr  error
	)
	for attempt := 0; attempt < ipAllowlistMaxAttempts; attempt++ {
		var bodyReader io.Reader
		if rawBody != nil {
			bodyReader = bytes.NewReader(rawBody)
		}
		resp, respBody, lastErr = ipAllowlistDoOnce(client, method, endpoint, bodyReader)
		// Retry only on 409 Conflict, which we know LD returns for
		// optimistic-concurrency races on the account allowlist
		// document. Everything else (transport errors, 4xx other
		// than 409, 5xx) falls back to the underlying retryablehttp
		// policy already applied by fallbackClient.
		if resp == nil || resp.StatusCode != http.StatusConflict {
			return resp, respBody, lastErr
		}
		if attempt == ipAllowlistMaxAttempts-1 {
			return resp, respBody, lastErr
		}
		time.Sleep(ipAllowlistBackoff(attempt))
	}
	return resp, respBody, lastErr
}

func ipAllowlistDoOnce(client *Client, method, endpoint string, body io.Reader) (*http.Response, []byte, error) {
	req, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", client.apiKey)
	req.Header.Set("LD-API-Version", "beta")

	var resp *http.Response
	err = client.withConcurrency(client.ctx, func() error {
		resp, err = client.fallbackClient.Do(req)
		return err
	})
	if err != nil {
		return resp, nil, fmt.Errorf("request failed: %w", err)
	}

	respBody, readErr := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if readErr != nil {
		return resp, nil, fmt.Errorf("failed to read response body: %w", readErr)
	}

	if resp.StatusCode >= 400 {
		var apiErr ipAllowlistErrorResponse
		if jsonErr := json.Unmarshal(respBody, &apiErr); jsonErr == nil && apiErr.Message != "" {
			return resp, respBody, fmt.Errorf("API error (%d): %s", resp.StatusCode, apiErr.Message)
		}
		return resp, respBody, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	return resp, respBody, nil
}

func getIpAllowlist(client *Client) (*ipAllowlistResponse, error) {
	_, respBody, err := ipAllowlistRequest(client, http.MethodGet, ipAllowlistBasePath, nil)
	if err != nil {
		return nil, err
	}

	var result ipAllowlistResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal IP allowlist response: %w", err)
	}
	return &result, nil
}

func createIpAllowlistEntry(client *Client, ipAddress string, description *string) (*ipAllowlistEntryResponse, error) {
	reqBody := createIpAllowlistEntryRequest{
		IpAddress:   ipAddress,
		Description: description,
	}

	_, respBody, err := ipAllowlistRequest(client, http.MethodPost, ipAllowlistBasePath, reqBody)
	if err != nil {
		return nil, err
	}

	var result ipAllowlistEntryResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal IP allowlist entry response: %w", err)
	}
	return &result, nil
}

func patchIpAllowlistEntry(client *Client, id string, description string) (*ipAllowlistEntryResponse, error) {
	path := fmt.Sprintf("%s/%s", ipAllowlistBasePath, id)
	reqBody := patchIpAllowlistEntryRequest{
		Description: description,
	}

	_, respBody, err := ipAllowlistRequest(client, http.MethodPatch, path, reqBody)
	if err != nil {
		return nil, err
	}

	var result ipAllowlistEntryResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal IP allowlist entry response: %w", err)
	}
	return &result, nil
}

func deleteIpAllowlistEntry(client *Client, id string) error {
	path := fmt.Sprintf("%s/%s", ipAllowlistBasePath, id)
	_, _, err := ipAllowlistRequest(client, http.MethodDelete, path, nil)
	return err
}

func patchIpAllowlistConfig(client *Client, sessionEnabled, scopedEnabled *bool) (*ipAllowlistResponse, error) {
	reqBody := patchIpAllowlistConfigRequest{
		SessionAllowlistEnabled:  sessionEnabled,
		ApiTokenAllowlistEnabled: scopedEnabled,
	}

	_, respBody, err := ipAllowlistRequest(client, http.MethodPatch, ipAllowlistBasePath, reqBody)
	if err != nil {
		return nil, err
	}

	var result ipAllowlistResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal IP allowlist response: %w", err)
	}
	return &result, nil
}

func findIpAllowlistEntryByID(entries []ipAllowlistEntryResponse, id string) *ipAllowlistEntryResponse {
	for _, entry := range entries {
		if entry.ID == id {
			return &entry
		}
	}
	return nil
}
