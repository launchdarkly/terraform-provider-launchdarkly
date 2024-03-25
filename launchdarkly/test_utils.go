package launchdarkly

import (
	"fmt"

	ldapi "github.com/launchdarkly/api-client-go/v15"
)

// testAccProjectScaffoldCreate creates a project with the given project parameters
func testAccProjectScaffoldCreate(client *Client, projectBody ldapi.ProjectPost) (*ldapi.Project, error) {
	_, _, err := client.ld.ProjectsApi.PostProject(client.ctx).ProjectPost(projectBody).Execute()
	if err != nil {
		return nil, err
	}
	project, _, err := client.ld.ProjectsApi.GetProject(client.ctx, projectBody.Key).Expand("environments").Execute()
	if err != nil {
		return nil, err
	}

	return project, nil
}

func testAccProjectScaffoldDelete(client *Client, projectKey string) error {
	_, err := client.ld.ProjectsApi.DeleteProject(client.ctx, projectKey).Execute()
	if err != nil {
		return err
	}
	return nil
}

func testAccFeatureFlagScaffold(client *Client, projectKey string, flagBody ldapi.FeatureFlagBody) (*ldapi.FeatureFlag, error) {
	projectBody := ldapi.ProjectPost{
		Name: "Flag Test Project",
		Key:  projectKey,
	}
	project, err := testAccProjectScaffoldCreate(client, projectBody)
	if err != nil {
		return nil, err
	}

	flag, _, err := client.ld.FeatureFlagsApi.PostFeatureFlag(client.ctx, project.Key).FeatureFlagBody(flagBody).Execute()
	if err != nil {
		return nil, err
	}
	return flag, nil

}

func addContextKindToProject(client *Client, projectKey string, contextKind string) error {
	hideInTargeting := false
	contextKindBody := *ldapi.NewUpsertContextKindPayload(contextKind)
	contextKindBody.HideInTargeting = &hideInTargeting

	_, _, err := client.ld.ContextsApi.PutContextKind(client.ctx, projectKey, contextKind).UpsertContextKindPayload(contextKindBody).Execute()
	if err != nil {
		return fmt.Errorf("failed to create context kind %s on project %s for test scaffolding: %s", contextKind, projectKey, err.Error())
	}
	return nil
}

func intfPtr(i interface{}) *interface{} {
	return &i
}
