package launchdarkly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func releasePolicyRead(ctx context.Context, d *schema.ResourceData, meta interface{}, isDataSource bool) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*Client)

	projectKey := d.Get(PROJECT_KEY).(string)
	policyKey := d.Get(KEY).(string)

	policy, res, err := getReleasePolicy(client, projectKey, policyKey)

	if isStatusNotFound(res) && !isDataSource {
		log.Printf("[WARN] failed to find release policy with key %q in project %q, removing from state if present", policyKey, projectKey)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("[WARN] failed to find release policy with key %q in project %q, removing from state if present", policyKey, projectKey),
		})
		d.SetId("")
		return diags
	}
	if err != nil {
		return diag.Errorf("failed to get release policy with key %q in project %q: %v", policyKey, projectKey, err)
	}

	_ = d.Set(PROJECT_KEY, policy.ProjectKey)
	_ = d.Set(KEY, policy.Key)
	_ = d.Set(NAME, policy.Name)
	_ = d.Set(RELEASE_METHOD, policy.ReleaseMethod)
	d.SetId(policy.Id)

	// Set scope if it exists
	if policy.Scope != nil {
		scopeList := []map[string]interface{}{
			{
				SCOPE_ENVIRONMENT_KEYS: policy.Scope.EnvironmentKeys,
			},
		}
		err = d.Set(SCOPE, scopeList)
		if err != nil {
			return diag.Errorf("could not set scope on release policy with key %q: %v", policy.Key, err)
		}
	}

	// Set guarded release config if it exists
	if policy.GuardedReleaseConfig != nil && policy.ReleaseMethod == "guarded-release" {
		configList := []map[string]interface{}{
			{
				ROLLBACK_ON_REGRESSION: policy.GuardedReleaseConfig.RollbackOnRegression,
			},
		}
		// This will always be a list of one
		if policy.GuardedReleaseConfig.MinSampleSize != nil && *policy.GuardedReleaseConfig.MinSampleSize > 0 {
			configList[0][MIN_SAMPLE_SIZE] = policy.GuardedReleaseConfig.MinSampleSize
		}
		if len(policy.GuardedReleaseConfig.MetricKeys) > 0 {
			configList[0][METRIC_KEYS] = policy.GuardedReleaseConfig.MetricKeys
		}
		if len(policy.GuardedReleaseConfig.MetricGroupKeys) > 0 {
			configList[0][METRIC_GROUP_KEYS] = policy.GuardedReleaseConfig.MetricGroupKeys
		}

		err = d.Set(GUARDED_RELEASE_CONFIG, configList)
		if err != nil {
			return diag.Errorf("could not set guarded_release_config on release policy with key %q: %v", policy.Key, err)
		}
	}

	return diags
}

// ReleasePolicy represents a release policy
type ReleasePolicy struct {
	Id                   string                `json:"id"`
	Key                  string                `json:"key"`
	Name                 string                `json:"name"`
	ProjectKey           string                `json:"projectKey"`
	ReleaseMethod        string                `json:"releaseMethod"`
	Scope                *ReleasePolicyScope   `json:"scope,omitempty"`
	GuardedReleaseConfig *GuardedReleaseConfig `json:"guardedReleaseConfig,omitempty"`
}

// ReleasePolicyScope represents the scope configuration for a release policy
type ReleasePolicyScope struct {
	EnvironmentKeys []string `json:"environmentKeys"`
}

// GuardedReleaseConfig represents the configuration for guarded release
type GuardedReleaseConfig struct {
	RollbackOnRegression bool     `json:"rollbackOnRegression"`
	MinSampleSize        *int     `json:"minSampleSize,omitempty"`
	MetricKeys           []string `json:"metricKeys,omitempty"`
	MetricGroupKeys      []string `json:"metricGroupKeys,omitempty"`
}

func getReleasePolicy(client *Client, projectKey, policyKey string) (*ReleasePolicy, *http.Response, error) {
	return getReleasePolicyRaw(client, projectKey, policyKey)
}

func buildReleasePolicyURL(client *Client, projectKey, policyKey string) string {
	host := client.apiHost
	if host == "" {
		host = "app.launchdarkly.com"
	}
	var url string
	if policyKey == "" {
		url = fmt.Sprintf("https://%s/api/v2/projects/%s/release-policies", host, projectKey)
	} else {
		url = fmt.Sprintf("https://%s/api/v2/projects/%s/release-policies/%s", host, projectKey, policyKey)
	}
	return url
}

func getReleasePolicyRaw(client *Client, projectKey, policyKey string) (*ReleasePolicy, *http.Response, error) {
	url := buildReleasePolicyURL(client, projectKey, policyKey)

	req, err := http.NewRequestWithContext(client.ctx, "GET", url, nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Authorization", client.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LD-API-Version", "beta")

	resp, err := client.ld.GetConfig().HTTPClient.Do(req)
	if err != nil {
		return nil, resp, err
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, resp, readErr
	}

	if resp.StatusCode >= 400 {
		return nil, resp, fmt.Errorf("LaunchDarkly API error - Status: %d, Response: %s", resp.StatusCode, string(body))
	}

	var policy ReleasePolicy
	if err := json.Unmarshal(body, &policy); err != nil {
		return nil, resp, err
	}
	policy.Id = fmt.Sprintf("%s/%s", projectKey, policyKey)
	policy.ProjectKey = projectKey

	return &policy, resp, nil
}

func createReleasePolicy(client *Client, projectKey string, policyPost map[string]interface{}) (*ReleasePolicy, error) {
	url := buildReleasePolicyURL(client, projectKey, "")
	jsonData, err := json.Marshal(policyPost)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(client.ctx, "POST", url, bytes.NewBuffer(jsonData))
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

	var policy ReleasePolicy
	if err := json.Unmarshal(body, &policy); err != nil {
		return nil, err
	}

	return &policy, nil
}

func putReleasePolicy(client *Client, projectKey, policyKey string, updatedPolicy map[string]interface{}) error {
	url := buildReleasePolicyURL(client, projectKey, policyKey)
	jsonData, err := json.Marshal(updatedPolicy)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(client.ctx, "PUT", url, bytes.NewBuffer(jsonData))
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

func deleteReleasePolicy(client *Client, projectKey, policyKey string) error {
	url := buildReleasePolicyURL(client, projectKey, policyKey)

	req, err := http.NewRequestWithContext(client.ctx, "DELETE", url, nil)
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
