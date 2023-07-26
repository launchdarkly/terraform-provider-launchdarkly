package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v12"
)

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

	return diags
}

func getFullProject(client *Client, projectKey string) (*ldapi.Project, *http.Response, error) {
	project, resp, err := client.ld.ProjectsApi.GetProject(client.ctx, projectKey).Execute()
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
		envPage, resp, err := client.ld.EnvironmentsApi.GetEnvironmentsByProject(
			client.ctx, projectKey).Limit(pageLimit).Offset(currentPage * pageLimit).Execute()
		if err != nil {
			return *ldapi.NewEnvironments(), resp, err
		}
		envItems = append(envItems, envPage.Items...)
		if len(envItems) >= int(envPage.GetTotalCount()) {
			allFetched = true
		}
	}

	envs := *ldapi.NewEnvironments()
	envs.SetItems(envItems)
	envs.SetTotalCount(int32(len(envItems)))
	return envs, nil, nil
}
