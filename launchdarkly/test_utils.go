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

func intfPtr(i interface{}) *interface{} {
	return &i
}
