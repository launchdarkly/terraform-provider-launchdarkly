package launchdarkly

import (
	ldapi "github.com/launchdarkly/api-client-go/v10"
)

// testAccDataSourceProjectCreate creates a project with the given project parameters
func testAccDataSourceProjectCreate(client *Client, projectBody ldapi.ProjectPost) (*ldapi.Project, error) {
	_, _, err := client.ld.ProjectsApi.PostProject(client.ctx).ProjectPost(projectBody).Execute()
	if err != nil {
		return nil, err
	}
	project, _, err := client.ld.ProjectsApi.GetProject(client.ctx, projectBody.Key).Expand("environments").Execute()

	return project, nil
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

	flag, _, err := client.ld.FeatureFlagsApi.PostFeatureFlag(client.ctx, project.Key).FeatureFlagBody(flagBody).Execute()
	if err != nil {
		return nil, err
	}
	return flag, nil

}

func intfPtr(i interface{}) *interface{} {
	return &i
}
