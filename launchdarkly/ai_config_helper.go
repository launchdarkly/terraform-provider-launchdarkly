package launchdarkly

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	ldapi "github.com/launchdarkly/api-client-go/v17"
)

func (c *Client) postAIConfig(projectKey string, aiConfig AIConfig) (*http.Response, *AIConfig, error) {
	url := fmt.Sprintf("%s/api/v2/projects/%s/ai-configs", c.apiHost, projectKey)
	body, err := json.Marshal(aiConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal AI Config: %s", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to prepare request: %s", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", c.apiKey)

	resp, err := c.fallbackClient.Do(req)
	if err != nil {
		return resp, nil, handleLdapiErr(err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("failed to read response body: %s", err)
	}

	if resp.StatusCode >= 400 {
		return resp, nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result AIConfig
	if err = json.Unmarshal(respBody, &result); err != nil {
		return resp, nil, fmt.Errorf("failed to unmarshal response: %s", err)
	}

	return resp, &result, nil
}

func (c *Client) getAIConfig(projectKey, configKey string) (*AIConfig, *http.Response, error) {
	url := fmt.Sprintf("%s/api/v2/projects/%s/ai-configs/%s", c.apiHost, projectKey, configKey)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to prepare request: %s", err)
	}

	req.Header.Add("Authorization", c.apiKey)

	resp, err := c.fallbackClient.Do(req)
	if err != nil {
		return nil, resp, handleLdapiErr(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, resp, nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to read response body: %s", err)
	}

	if resp.StatusCode >= 400 {
		return nil, resp, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result AIConfig
	if err = json.Unmarshal(respBody, &result); err != nil {
		return nil, resp, fmt.Errorf("failed to unmarshal response: %s", err)
	}

	return &result, resp, nil
}

func (c *Client) patchAIConfig(projectKey, configKey string, patch []ldapi.PatchOperation) (*http.Response, error) {
	url := fmt.Sprintf("%s/api/v2/projects/%s/ai-configs/%s", c.apiHost, projectKey, configKey)
	patchWithComment := ldapi.PatchWithComment{
		Patch: patch,
	}

	body, err := json.Marshal(patchWithComment)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal patch: %s", err)
	}

	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to prepare request: %s", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", c.apiKey)

	resp, err := c.fallbackClient.Do(req)
	if err != nil {
		return resp, handleLdapiErr(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return resp, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return resp, nil
}

func (c *Client) deleteAIConfig(projectKey, configKey string) (*http.Response, error) {
	url := fmt.Sprintf("%s/api/v2/projects/%s/ai-configs/%s", c.apiHost, projectKey, configKey)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare request: %s", err)
	}

	req.Header.Add("Authorization", c.apiKey)

	resp, err := c.fallbackClient.Do(req)
	if err != nil {
		return resp, handleLdapiErr(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
		respBody, _ := io.ReadAll(resp.Body)
		return resp, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return resp, nil
}
