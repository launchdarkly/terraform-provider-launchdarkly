package launchdarkly

import (
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
)

func projectRead(d *schema.ResourceData, meta interface{}, isDataSource bool) error {
	client := meta.(*Client)
	projectKey := d.Get(KEY).(string)

	rawProject, res, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.ProjectsApi.GetProject(client.ctx, projectKey)
	})
	// return nil error for resource reads but 404 for data source reads
	if isStatusNotFound(res) && !isDataSource {
		log.Printf("[WARN] failed to find project with key %q, removing from state if present", projectKey)
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get project with key %q: %v", projectKey, err)
	}

	project := rawProject.(ldapi.Project)
	// the Id needs to be set on reads for the data source, but it will mess up the state for resource reads
	if isDataSource {
		d.SetId(project.Id)
	}
	_ = d.Set(KEY, project.Key)
	_ = d.Set(NAME, project.Name)

	// Only allow nested environments for the launchdarkly_project resource. The dedicated environment data source
	// should be used if a data source is required for a LaunchDarkly environment
	if !isDataSource {
		// Convert the returned environment list to a map so we can lookup each environment by key while preserving the
		// order defined in the config
		envMap := environmentsToResourceDataMap(project.Environments)

		// iterate over the environment keys in the order defined by the config and look up the environment returned by
		// LD's API
		rawEnvs := d.Get(ENVIRONMENTS).([]interface{})
		envConfigKeys := rawEnvironmentConfigsToKeyList(rawEnvs)
		environments := make([]interface{}, 0, len(envConfigKeys))
		for _, envKey := range envConfigKeys {
			environments = append(environments, envMap[envKey])
		}

		err = d.Set(ENVIRONMENTS, environments)
		if err != nil {
			return fmt.Errorf("could not set environments on project with key %q: %v", project.Key, err)
		}
	}

	err = d.Set(TAGS, project.Tags)
	if err != nil {
		return fmt.Errorf("could not set tags on project with key %q: %v", project.Key, err)
	}
	if isDataSource {
		defaultCSA := *project.DefaultClientSideAvailability
		clientSideAvailability := map[string]string{
			"using_environment_id": fmt.Sprintf("%v", defaultCSA.UsingEnvironmentId),
			"using_mobile_key":     fmt.Sprintf("%v", defaultCSA.UsingMobileKey),
		}
		err = d.Set(CLIENT_SIDE_AVAILABILITY, clientSideAvailability)
		if err != nil {
			return fmt.Errorf("could not set client_side_availability on project with key %q: %v", project.Key, err)
		}
	} else {
		err = d.Set(INCLUDE_IN_SNIPPET, project.IncludeInSnippetByDefault)
		if err != nil {
			return fmt.Errorf("could not set include_in_snippet on project with key %q: %v", project.Key, err)
		}
	}
	return nil
}
