package launchdarkly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// SegmentBodyWithViewKeys represents the segment creation request body with view_keys support.
// This is needed because the API client doesn't include the viewKeys field (it's hidden in the API spec).
type SegmentBodyWithViewKeys struct {
	Name                 string   `json:"name"`
	Key                  string   `json:"key"`
	Description          string   `json:"description,omitempty"`
	Tags                 []string `json:"tags,omitempty"`
	Unbounded            bool     `json:"unbounded,omitempty"`
	UnboundedContextKind string   `json:"unboundedContextKind,omitempty"`
	ViewKeys             []string `json:"viewKeys,omitempty"`
}

// createSegmentWithViewKeys creates a segment using a raw HTTP call to support the viewKeys field.
// This is necessary because the API client doesn't include viewKeys in SegmentBody.
func createSegmentWithViewKeys(ctx context.Context, client *Client, projectKey, envKey string, body SegmentBodyWithViewKeys) error {
	host := client.apiHost
	if host == "" {
		host = DEFAULT_LAUNCHDARKLY_HOST
	}

	var endpoint string
	if u, err := url.Parse(host); err == nil && u.Scheme != "" {
		u.Path = fmt.Sprintf("/api/v2/segments/%s/%s", projectKey, envKey)
		endpoint = u.String()
	} else {
		endpoint = fmt.Sprintf("https://%s/api/v2/segments/%s/%s", host, projectKey, envKey)
	}

	jsonData, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", client.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LD-API-Version", APIVersion)
	req.Header.Set("User-Agent", fmt.Sprintf("launchdarkly-terraform-provider/%s", version))

	resp, err := client.ld.GetConfig().HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return readErr
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("%d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(respBody))
	}

	return nil
}
