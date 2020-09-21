package launchdarkly

import (
	"fmt"
	"net/http"

	ldapi "github.com/launchdarkly/api-client-go"
)

// testAccDataSourceProjectCreate creates a project with the given project parameters
func testAccDataSourceProjectCreate(client *Client, projectBody ldapi.ProjectBody) (*ldapi.Project, error) {
	project, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.ProjectsApi.PostProject(client.ctx, projectBody)
	})
	if err != nil {
		return nil, err
	}
	if project, ok := project.(ldapi.Project); ok {
		return &project, nil
	}
	return nil, fmt.Errorf("failed to create project")
}

func testAccDataSourceProjectDelete(client *Client, projectKey string) error {
	_, err := client.ld.ProjectsApi.DeleteProject(client.ctx, projectKey)
	if err != nil {
		return err
	}
	return nil
}

// testAccDataSourceEnvironmentScaffold creates a project with the given projectKey with the given env params
// for environment data source tests
func testAccDataSourceEnvironmentScaffold(client *Client, projectKey string, envBody ldapi.EnvironmentPost) (*ldapi.Environment, error) {
	// create project
	projectBody := ldapi.ProjectBody{
		Name: "Env Test Project",
		Key:  projectKey,
		Environments: []ldapi.EnvironmentPost{
			envBody,
		},
	}
	project, err := testAccDataSourceProjectCreate(client, projectBody)
	if err != nil {
		return nil, err
	}
	for _, env := range project.Environments {
		if env.Key == envBody.Key {
			return &env, nil
		}
	}
	return nil, fmt.Errorf("failed to create env")
}

func testAccDataSourceFeatureFlagScaffold(client *Client, projectKey string, flagBody ldapi.FeatureFlagBody) (*ldapi.FeatureFlag, error) {
	projectBody := ldapi.ProjectBody{
		Name: "Flag Test Project",
		Key:  projectKey,
	}
	project, err := testAccDataSourceProjectCreate(client, projectBody)
	if err != nil {
		return nil, err
	}

	flag, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.FeatureFlagsApi.PostFeatureFlag(client.ctx, project.Key, flagBody, nil)
	})
	if flag, ok := flag.(ldapi.FeatureFlag); ok {
		return &flag, nil
	}
	return nil, fmt.Errorf("failed to create flag")
}

func testAccDataSourceScaffoldTeardown(client *Client, projectKey string) error {
	return testAccDataSourceProjectDelete(client, projectKey)
}
