package launchdarkly

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	ldapi "github.com/launchdarkly/api-client-go/v17"
)

func (c *Client) postAIConfig(projectKey string, aiConfig AIConfig) (*http.Response, *AIConfig, error) {
	url := fmt.Sprintf("/api/v2/projects/%s/ai-configs", projectKey)
	body, err := json.Marshal(aiConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal AI Config: %s", err)
	}

	req, err := c.ld.GetConfig().APIClient.PreparePost(c.ld.GetConfig().BasePath + url)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to prepare request: %s", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Body = body

	resp, err := c.ld.GetConfig().APIClient.CallAPI(req)
	if err != nil {
		return resp, nil, handleLdapiErr(err)
	}

	var result AIConfig
	if err = json.Unmarshal(resp.Body, &result); err != nil {
		return resp, nil, fmt.Errorf("failed to unmarshal response: %s", err)
	}

	return resp, &result, nil
}

func (c *Client) getAIConfig(projectKey, configKey string) (*AIConfig, *http.Response, error) {
	url := fmt.Sprintf("/api/v2/projects/%s/ai-configs/%s", projectKey, configKey)

	req, err := c.ld.GetConfig().APIClient.PrepareGet(c.ld.GetConfig().BasePath + url)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to prepare request: %s", err)
	}

	resp, err := c.ld.GetConfig().APIClient.CallAPI(req)
	if err != nil {
		return nil, resp, handleLdapiErr(err)
	}

	var result AIConfig
	if err = json.Unmarshal(resp.Body, &result); err != nil {
		return nil, resp, fmt.Errorf("failed to unmarshal response: %s", err)
	}

	return &result, resp, nil
}

func (c *Client) patchAIConfig(projectKey, configKey string, patch []ldapi.PatchOperation) (*http.Response, error) {
	url := fmt.Sprintf("/api/v2/projects/%s/ai-configs/%s", projectKey, configKey)
	patchWithComment := ldapi.PatchWithComment{
		Patch: patch,
	}

	body, err := json.Marshal(patchWithComment)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal patch: %s", err)
	}

	req, err := c.ld.GetConfig().APIClient.PreparePatch(c.ld.GetConfig().BasePath + url)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare request: %s", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Body = body

	resp, err := c.ld.GetConfig().APIClient.CallAPI(req)
	if err != nil {
		return resp, handleLdapiErr(err)
	}

	return resp, nil
}

func (c *Client) deleteAIConfig(projectKey, configKey string) (*http.Response, error) {
	url := fmt.Sprintf("/api/v2/projects/%s/ai-configs/%s", projectKey, configKey)

	req, err := c.ld.GetConfig().APIClient.PrepareDelete(c.ld.GetConfig().BasePath + url)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare request: %s", err)
	}

	resp, err := c.ld.GetConfig().APIClient.CallAPI(req)
	if err != nil {
		return resp, handleLdapiErr(err)
	}

	return resp, nil
}

func getRandomSleepDuration(duration time.Duration) time.Duration {
	if duration <= 0 {
		return 0
	}
	return time.Duration(rand.Int63n(int64(duration)))
}
