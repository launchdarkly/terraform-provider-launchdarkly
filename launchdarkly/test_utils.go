package launchdarkly

import (
	"fmt"
	"net/http"

	ldapi "github.com/launchdarkly/api-client-go/v7"
)

// testAccDataSourceProjectCreate creates a project with the given project parameters
func testAccDataSourceProjectCreate(client *Client, projectBody ldapi.ProjectPost) (*ldapi.Project, error) {
	project, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.ProjectsApi.PostProject(client.ctx).ProjectPost(projectBody).Execute()
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
	_, err := client.ld.ProjectsApi.DeleteProject(client.ctx, projectKey).Execute()
	if err != nil {
		return err
	}
	return nil
}

func testAccDataSourceFeatureFlagScaffold(client *Client, projectKey string, flagBody ldapi.FeatureFlagBody) (*ldapi.FeatureFlag, error) {
	projectBody := ldapi.ProjectPost{
		Name: "Flag Test Project",
		Key:  projectKey,
	}
	project, err := testAccDataSourceProjectCreate(client, projectBody)
	if err != nil {
		return nil, err
	}

	flag, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.FeatureFlagsApi.PostFeatureFlag(client.ctx, project.Key).FeatureFlagBody(flagBody).Execute()
	})
	if err != nil {
		return nil, err
	}
	if flag, ok := flag.(ldapi.FeatureFlag); ok {
		return &flag, nil
	}
	return nil, fmt.Errorf("failed to create flag")

}

func intfPtr(i interface{}) *interface{} {
	return &i
}
