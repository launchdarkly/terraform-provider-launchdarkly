package launchdarkly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	ldapi "github.com/launchdarkly/api-client-go/v22"
)

// buildProjectURL constructs a properly formatted URL for the projects API endpoint.
// It handles cases where apiHost may or may not include a scheme.
func buildProjectURL(apiHost, projectKey string) string {
	host := apiHost
	if host == "" {
		host = DEFAULT_LAUNCHDARKLY_HOST
	}

	if u, err := url.Parse(host); err == nil && u.Scheme != "" {
		u.Path = fmt.Sprintf("/api/v2/projects/%s", projectKey)
		return u.String()
	}

	return fmt.Sprintf("https://%s/api/v2/projects/%s", host, projectKey)
}

// projectExists reports whether a project with the given key exists.
// Used by framework feature_flag / segment / FFE / metric resources.
func projectExists(projectKey string, client *Client) (bool, error) {
	var res *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, res, err = client.ld.ProjectsApi.GetProject(client.ctx, projectKey).Execute()
		return err
	})
	if isStatusNotFound(res) {
		log.Println("got 404 when getting project. returning false.")
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get project with key %q: %v", projectKey, handleLdapiErr(err))
	}
	return true, nil
}

func getFullProject(client *Client, projectKey string) (*ldapi.Project, *http.Response, error) {
	var project *ldapi.Project
	var resp *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		project, resp, err = client.ld.ProjectsApi.GetProject(client.ctx, projectKey).Execute()
		return err
	})
	if err != nil {
		return project, resp, err
	}

	envs, resp, err := getAllEnvironments(client, projectKey)
	if err != nil {
		return project, resp, err
	}

	project.Environments = &envs
	return project, resp, nil
}

func getAllEnvironments(client *Client, projectKey string) (ldapi.Environments, *http.Response, error) {
	envItems := make([]ldapi.Environment, 0)
	pageLimit := int64(20)
	allFetched := false
	for currentPage := int64(0); !allFetched; currentPage++ {
		var envPage *ldapi.Environments
		var resp *http.Response
		var err error
		err = client.withConcurrency(client.ctx, func() error {
			envPage, resp, err = client.ld.EnvironmentsApi.GetEnvironmentsByProject(
				client.ctx, projectKey).Limit(pageLimit).Offset(currentPage * pageLimit).Execute()
			return err
		})
		if err != nil {
			return *ldapi.NewEnvironments(envItems), resp, err
		}
		envItems = append(envItems, envPage.Items...)
		if len(envItems) >= int(envPage.GetTotalCount()) {
			allFetched = true
		}
	}

	envs := *ldapi.NewEnvironments(envItems)
	envs.SetTotalCount(int32(len(envItems)))
	return envs, nil, nil
}

// ProjectViewSettings represents the view association requirement settings for a project.
// These fields are not yet in the official API client, so we use raw HTTP to read them.
type ProjectViewSettings struct {
	RequireViewAssociationForNewFlags    bool `json:"requireViewAssociationForNewFlags"`
	RequireViewAssociationForNewSegments bool `json:"requireViewAssociationForNewSegments"`
}

// getProjectViewSettings fetches the view association requirement settings for a project.
// Since these fields are not in the official API client model, we make a raw HTTP request
// and parse only the fields we need.
func getProjectViewSettings(ctx context.Context, client *Client, projectKey string) (*ProjectViewSettings, error) {
	endpoint := buildProjectURL(client.apiHost, projectKey)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", client.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LD-API-Version", APIVersion)
	req.Header.Set("User-Agent", fmt.Sprintf("launchdarkly-terraform-provider/%s", version))

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

	var settings ProjectViewSettings
	if err := json.Unmarshal(body, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

// patchProjectViewSettings updates the view association requirement settings for a project.
func patchProjectViewSettings(ctx context.Context, client *Client, projectKey string, flagsRequired, segmentsRequired bool, flagsChanged, segmentsChanged bool) error {
	endpoint := buildProjectURL(client.apiHost, projectKey)

	var patchOps []map[string]interface{}
	if flagsChanged {
		patchOps = append(patchOps, map[string]interface{}{
			"op":    "replace",
			"path":  "/requireViewAssociationForNewFlags",
			"value": flagsRequired,
		})
	}
	if segmentsChanged {
		patchOps = append(patchOps, map[string]interface{}{
			"op":    "replace",
			"path":  "/requireViewAssociationForNewSegments",
			"value": segmentsRequired,
		})
	}

	if len(patchOps) == 0 {
		return nil
	}

	jsonData, err := json.Marshal(patchOps)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", endpoint, bytes.NewBuffer(jsonData))
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

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return readErr
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("LaunchDarkly API error - Status: %d, Response: %s", resp.StatusCode, string(body))
	}

	return nil
}
