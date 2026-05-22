package launchdarkly

// feature_flag_helpers.go holds shared helpers used by
// resource_feature_flag_framework.go and
// resource_feature_flag_environment_framework.go.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	ldapi "github.com/launchdarkly/api-client-go/v22"
)

// Variation type identifiers.
const (
	BOOL_VARIATION   = "boolean"
	STRING_VARIATION = "string"
	NUMBER_VARIATION = "number"
	JSON_VARIATION   = "json"
)

// Custom property limits used by the framework schema validators.
const (
	CUSTOM_PROPERTY_CHAR_LIMIT = 64
	CUSTOM_PROPERTY_ITEM_LIMIT = 64
)

// flagIdToKeys parses a `project_key/flag_key` composite id.
func flagIdToKeys(id string) (projectKey, flagKey string, err error) {
	if strings.Count(id, "/") != 1 {
		return "", "", fmt.Errorf("found unexpected flag id format: %q expected format: 'project_key/flag_key'", id)
	}
	parts := strings.SplitN(id, "/", 2)
	return parts[0], parts[1], nil
}

// getProjectDefaultCSAandIncludeInSnippet returns the project's default
// CSA + IncludeInSnippet for use when a feature_flag config omits both.
func getProjectDefaultCSAandIncludeInSnippet(client *Client, projectKey string) (ldapi.ClientSideAvailability, bool, error) {
	var project *ldapi.Project
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		project, _, err = client.ld.ProjectsApi.GetProject(client.ctx, projectKey).Execute()
		return err
	})
	if err != nil {
		return ldapi.ClientSideAvailability{}, false, err
	}
	return *project.DefaultClientSideAvailability, project.IncludeInSnippetByDefault, nil
}

// FeatureFlagBodyWithViewKeys represents the feature flag creation
// request body with view_keys support. The generated API client doesn't
// include viewKeys, so we use raw HTTP for this path.
type FeatureFlagBodyWithViewKeys struct {
	Name                   string                            `json:"name"`
	Key                    string                            `json:"key"`
	Description            string                            `json:"description,omitempty"`
	Variations             []ldapi.Variation                 `json:"variations,omitempty"`
	Temporary              bool                              `json:"temporary,omitempty"`
	Tags                   []string                          `json:"tags,omitempty"`
	Defaults               *ldapi.Defaults                   `json:"defaults,omitempty"`
	ClientSideAvailability *ldapi.ClientSideAvailabilityPost `json:"clientSideAvailability,omitempty"`
	ViewKeys               []string                          `json:"viewKeys,omitempty"`
}

func createFeatureFlagWithViewKeys(ctx context.Context, client *Client, projectKey string, body FeatureFlagBodyWithViewKeys) error {
	host := client.apiHost
	if host == "" {
		host = DEFAULT_LAUNCHDARKLY_HOST
	}
	var endpoint string
	if u, err := url.Parse(host); err == nil && u.Scheme != "" {
		u.Path = fmt.Sprintf("/api/v2/flags/%s", projectKey)
		endpoint = u.String()
	} else {
		endpoint = fmt.Sprintf("https://%s/api/v2/flags/%s", host, projectKey)
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

// stringifyValue converts a basic Go value (the json-decoded interface)
// to its canonical string form. Used by clauses and variations.
func stringifyValue(value interface{}) string {
	switch v := value.(type) {
	case int:
		return strconv.Itoa(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case string:
		return v
	}
	return ""
}

// variationValueToString stringifies a *interface{} typed variation
// value according to its LD variation type (json variations are
// normalised via NormalizeJsonString to avoid whitespace drift).
func variationValueToString(value *interface{}, variationType string) (string, error) {
	if variationType != JSON_VARIATION {
		return stringifyValue(*value), nil
	}
	byteVal, err := json.Marshal(*value)
	if err != nil {
		return "", fmt.Errorf("unable to marshal json variation value: %v", err)
	}
	return normalizeJSONString(string(byteVal))
}

// variationsToVariationType infers the variation type from the first
// element of a variations slice.
func variationsToVariationType(variations []ldapi.Variation) (string, error) {
	if len(variations) == 0 {
		return "", fmt.Errorf("variations slice is empty")
	}
	value := variations[0].Value
	if value == nil {
		return "", fmt.Errorf("nil variation value: %v", value)
	}
	switch value.(type) {
	case bool:
		return BOOL_VARIATION, nil
	case string:
		return STRING_VARIATION, nil
	case float64:
		return NUMBER_VARIATION, nil
	case map[string]interface{}, []interface{}:
		return JSON_VARIATION, nil
	default:
		return "", fmt.Errorf("unknown variation type: %T", value)
	}
}

// normalizeJSONString re-marshals a json string with key order
// preserved by the encoder default.
func normalizeJSONString(input string) (string, error) {
	var v interface{}
	if err := json.Unmarshal([]byte(input), &v); err != nil {
		return "", err
	}
	out, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// getDependentFlags returns the flags that reference the named flag as a
// prerequisite, across every environment in the project. Backs the
// plan-time destroy check in FeatureFlagResource.ModifyPlan so the
// "Flag is still in use as a prerequisite" 409 from DELETE surfaces at
// plan time instead of apply time.
//
// The endpoint requires LD-API-Version: beta.
// Enterprise-only; non-Enterprise tokens get a 403 which the caller
// should degrade to a warning so apply behaviour is unchanged.
func getDependentFlags(ctx context.Context, client *Client, projectKey, flagKey string) (*ldapi.MultiEnvironmentDependentFlags, error) {
	deps, _, _, err := fetchDependentFlags(ctx, client, projectKey, flagKey)
	return deps, err
}

// fetchDependentFlags issues the dependent-flags beta request via raw
// HTTP so we can set LD-API-Version: beta explicitly even for generated
// client methods that do not expose an LDAPIVersion(...) setter.
// Returns parsed body on success, plus response status/body for richer
// diagnostics in callers (especially acceptance tests).
func fetchDependentFlags(ctx context.Context, client *Client, projectKey, flagKey string) (*ldapi.MultiEnvironmentDependentFlags, int, string, error) {
	host := client.apiHost
	if host == "" {
		host = DEFAULT_LAUNCHDARKLY_HOST
	}

	escapedProjectKey := url.PathEscape(projectKey)
	escapedFlagKey := url.PathEscape(flagKey)
	var endpoint string
	if u, err := url.Parse(host); err == nil && u.Scheme != "" {
		u.Path = fmt.Sprintf("/api/v2/flags/%s/%s/dependent-flags", escapedProjectKey, escapedFlagKey)
		endpoint = u.String()
	} else {
		endpoint = fmt.Sprintf("https://%s/api/v2/flags/%s/%s/dependent-flags", host, escapedProjectKey, escapedFlagKey)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, "", err
	}
	req.Header.Set("Authorization", client.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LD-API-Version", "beta")
	req.Header.Set("User-Agent", fmt.Sprintf("launchdarkly-terraform-provider/%s", version))

	var resp *http.Response
	err = client.withConcurrency(ctx, func() error {
		var reqErr error
		resp, reqErr = client.ld.GetConfig().HTTPClient.Do(req)
		return reqErr
	})
	if err != nil {
		return nil, 0, "", err
	}
	defer resp.Body.Close()

	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, resp.StatusCode, "", readErr
	}
	bodyStr := string(respBody)
	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, bodyStr, fmt.Errorf("%d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), bodyStr)
	}

	var deps ldapi.MultiEnvironmentDependentFlags
	if err := json.Unmarshal(respBody, &deps); err != nil {
		return nil, resp.StatusCode, bodyStr, fmt.Errorf("failed to decode dependent flags response: %w", err)
	}
	return &deps, resp.StatusCode, bodyStr, nil
}

// formatDependentFlagsHint renders the diagnostic detail body for the
// plan-time prerequisite-delete error. One bullet per (env, flag) pair,
// followed by a remediation pointer.
func formatDependentFlagsHint(items []ldapi.MultiEnvironmentDependentFlag) string {
	var b strings.Builder
	b.WriteString("The following flags reference this flag as a prerequisite:\n")
	for _, item := range items {
		for _, env := range item.Environments {
			fmt.Fprintf(&b, "  - environment %q, flag %q\n", env.Key, item.Key)
		}
	}
	b.WriteString("\nRemove the prerequisite from each listed flag (edit its launchdarkly_feature_flag_environment.prerequisites block) before destroying this flag.")
	return b.String()
}
