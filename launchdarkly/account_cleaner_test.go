package launchdarkly

import (
	"os"
	"testing"

	"github.com/launchdarkly/api-client-go"
	"github.com/stretchr/testify/require"
)

func TestCleanAccount(t *testing.T) {
	t.SkipNow()
	apiKey := os.Getenv(launchDarklyApiKeyEnvVar)
	require.NotEmpty(t, apiKey)

	c := NewClient(apiKey)
	// make sure we have a dummy project
	_, response, err := c.ld.ProjectsApi.GetProject(c.ctx, "dummy-project")

	if response.StatusCode == 404 {
		_, _, err = c.ld.ProjectsApi.PostProject(c.ctx, ldapi.ProjectBody{Name: "dummy-project", Key: "dummy-project"})
		require.NoError(t, err)
	} else {
		require.NoError(t, err)
	}
	projects, _, err := c.ld.ProjectsApi.GetProjects(c.ctx)
	require.NoError(t, err)

	// delete all but dummy project
	for _, p := range projects.Items {
		if p.Key != "dummy-project" {
			_, err := c.ld.ProjectsApi.DeleteProject(c.ctx, p.Key)
			require.NoError(t, err)
		}
	}
}
