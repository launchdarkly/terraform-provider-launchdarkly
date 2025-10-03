package launchdarkly

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// API functions for release policy order management
func getReleasePolicyOrder(client *Client, projectKey string) ([]string, error) {
	url := buildReleasePolicyURL(client, projectKey, "")

	req, err := http.NewRequestWithContext(client.ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", client.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LD-API-Version", "beta")

	resp, err := client.ld.GetConfig().HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, readErr
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("LaunchDarkly API error - Status: %d, Response: %s", resp.StatusCode, string(body))
	}

	orderResponse := struct {
		Items []ReleasePolicy `json:"items"`
	}{}
	if err := json.Unmarshal(body, &orderResponse); err != nil {
		return nil, err
	}
	// We don't know if there's a default policy which will be omitted, so we're not sure what length to `make()` this
	policyKeys := []string{}
	for _, policy := range orderResponse.Items {
		if policy.Scope != nil {
			policyKeys = append(policyKeys, policy.Key)
		}
	}

	return policyKeys, nil
}

func updateReleasePolicyOrder(client *Client, projectKey string, policyKeys []string) error {
	url := buildReleasePolicyOrderURL(client, projectKey)

	jsonData, err := json.Marshal(policyKeys)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(client.ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", client.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LD-API-Version", "beta")

	resp, err := client.ld.GetConfig().HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return readErr
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("LaunchDarkly API error - Status: %d, Response: %s", resp.StatusCode, string(body))
	}

	return nil
}

func buildReleasePolicyOrderURL(client *Client, projectKey string) string {
	host := client.apiHost
	if host == "" {
		host = "app.launchdarkly.com"
	}
	return fmt.Sprintf("https://%s/api/v2/projects/%s/release-policies/order", host, projectKey)
}
