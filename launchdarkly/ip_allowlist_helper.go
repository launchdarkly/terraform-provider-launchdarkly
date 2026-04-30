package launchdarkly

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

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
	endpoint := fmt.Sprintf("%s%s", client.apiHost, path)
	if !strings.HasPrefix(endpoint, "http") {
		endpoint = "https://" + endpoint
	}

	var bodyReader io.Reader
	if body != nil {
		rawBody, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(rawBody)
	}

	req, err := http.NewRequest(method, endpoint, bodyReader)
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

	respBody, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return resp, nil, fmt.Errorf("failed to read response body: %w", err)
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
