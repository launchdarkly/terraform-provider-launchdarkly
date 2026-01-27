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

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v17"
)

// getDefaultAPIHost extracts the hostname from DEFAULT_LAUNCHDARKLY_HOST
func getDefaultAPIHost() string {
	u, _ := url.Parse(DEFAULT_LAUNCHDARKLY_HOST)
	return u.Host
}

func projectRead(ctx context.Context, d *schema.ResourceData, meta interface{}, isDataSource bool) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*Client)
	projectKey := d.Get(KEY).(string)

	project, res, err := getFullProject(client, projectKey)

	// return nil error for resource reads but 404 for data source reads
	if isStatusNotFound(res) && !isDataSource {
		log.Printf("[WARN] failed to find project with key %q, removing from state if present", projectKey)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("[WARN] failed to find project with key %q, removing from state if present", projectKey),
		})
		d.SetId("")
		return diags
	}
	if err != nil {
		return diag.Errorf("failed to get project with key %q: %v", projectKey, err)
	}

	defaultCSA := *project.DefaultClientSideAvailability
	clientSideAvailability := []map[string]interface{}{{
		"using_environment_id": defaultCSA.UsingEnvironmentId,
		"using_mobile_key":     defaultCSA.UsingMobileKey,
	}}
	// the Id and deprecated client_side_availability need to be set on reads for the data source, but it will mess up the state for resource reads
	if isDataSource {
		d.SetId(project.Id)
		err = d.Set(CLIENT_SIDE_AVAILABILITY, clientSideAvailability)
		if err != nil {
			return diag.Errorf("could not set client_side_availability on project with key %q: %v", project.Key, err)
		}
	}
	_ = d.Set(KEY, project.Key)
	_ = d.Set(NAME, project.Name)

	// Only allow nested environments for the launchdarkly_project resource. The dedicated environment data source
	// should be used if a data source is required for a LaunchDarkly environment
	if !isDataSource {
		// Convert the returned environment list to a map so we can lookup each environment by key while preserving the
		// order defined in the config
		envMap := environmentsToResourceDataMap(project.Environments.Items)

		// iterate over the environment keys in the order defined by the config and look up the environment returned by
		// LD's API
		rawEnvs := d.Get(ENVIRONMENTS).([]interface{})

		envConfigKeys := rawEnvironmentConfigsToKeyList(rawEnvs)
		envAddedMap := make(map[string]bool, len(project.Environments.Items))
		environments := make([]interface{}, 0, len(envConfigKeys))
		for _, envKey := range envConfigKeys {
			environments = append(environments, envMap[envKey])
			envAddedMap[envKey] = true
		}

		// Now add all environments that are not specified in the config.
		// This is required in order to successfully import nested environments because rawEnvs is always an empty slice
		// durning import, even if nested environments are defined in the config.
		for _, env := range project.Environments.Items {
			alreadyAdded := envAddedMap[env.Key]
			if !alreadyAdded {
				environments = append(environments, envMap[env.Key])
				envAddedMap[env.Key] = true
			}
		}

		err = d.Set(ENVIRONMENTS, environments)
		if err != nil {
			return diag.Errorf("could not set environments on project with key %q: %v", project.Key, err)
		}

		err = d.Set(INCLUDE_IN_SNIPPET, project.IncludeInSnippetByDefault)
		if err != nil {
			return diag.Errorf("could not set include_in_snippet on project with key %q: %v", project.Key, err)
		}
	}

	err = d.Set(TAGS, project.Tags)
	if err != nil {
		return diag.Errorf("could not set tags on project with key %q: %v", project.Key, err)
	}

	err = d.Set(DEFAULT_CLIENT_SIDE_AVAILABILITY, clientSideAvailability)
	if err != nil {
		return diag.Errorf("could not set default_client_side_availability on project with key %q: %v", project.Key, err)
	}

	// Fetch and set view association requirement fields using raw HTTP
	// These fields are not in the official API client model yet
	viewSettings, viewSettingsErr := getProjectViewSettings(client, projectKey)
	if viewSettingsErr != nil {
		// Log warning but don't fail the read - these fields may not be available on all accounts
		log.Printf("[WARN] failed to get view association settings for project %q: %v", projectKey, viewSettingsErr)
	} else {
		err = d.Set(REQUIRE_VIEW_ASSOCIATION_FOR_NEW_FLAGS, viewSettings.RequireViewAssociationForNewFlags)
		if err != nil {
			return diag.Errorf("could not set require_view_association_for_new_flags on project with key %q: %v", project.Key, err)
		}
		err = d.Set(REQUIRE_VIEW_ASSOCIATION_FOR_NEW_SEGMENTS, viewSettings.RequireViewAssociationForNewSegments)
		if err != nil {
			return diag.Errorf("could not set require_view_association_for_new_segments on project with key %q: %v", project.Key, err)
		}
	}

	return diags
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
func getProjectViewSettings(client *Client, projectKey string) (*ProjectViewSettings, error) {
	host := client.apiHost
	if host == "" {
		host = getDefaultAPIHost()
	}
	url := fmt.Sprintf("https://%s/api/v2/projects/%s", host, projectKey)

	req, err := http.NewRequestWithContext(client.ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", client.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LD-API-Version", APIVersion)

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
func patchProjectViewSettings(client *Client, projectKey string, flagsRequired, segmentsRequired bool, flagsChanged, segmentsChanged bool) error {
	host := client.apiHost
	if host == "" {
		host = getDefaultAPIHost()
	}
	url := fmt.Sprintf("https://%s/api/v2/projects/%s", host, projectKey)

	// Build patch operations only for fields that changed
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

	req, err := http.NewRequestWithContext(client.ctx, "PATCH", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", client.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LD-API-Version", APIVersion)

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
