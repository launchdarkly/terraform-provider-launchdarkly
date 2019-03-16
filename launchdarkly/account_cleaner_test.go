package launchdarkly

import (
	"context"
	"os"
	"testing"

	ldapi "github.com/launchdarkly/api-client-go"
	"github.com/stretchr/testify/require"
)

func TestCleanAccount(t *testing.T) {
	t.SkipNow()
	apiKey := os.Getenv(launchDarklyApiKeyEnvVar)
	require.NotEmpty(t, apiKey)

	ctx := context.WithValue(context.Background(), ldapi.ContextAPIKey, ldapi.APIKey{
		Key: apiKey,
	})

	client := ldapi.NewAPIClient(ldapi.NewConfiguration())

	// make sure we have a dummy project
	_, response, err := client.ProjectsApi.GetProject(ctx, "dummy-project")

	if response.StatusCode == 404 {
		_, _, err = client.ProjectsApi.PostProject(ctx, ldapi.ProjectBody{Name: "dummy-project", Key: "dummy-project"})
		require.NoError(t, err)
	} else {
		require.NoError(t, err)
	}
	projects, _, err := client.ProjectsApi.GetProjects(ctx)
	require.NoError(t, err)

	// delete all but dummy project
	for _, p := range projects.Items {
		if p.Key != "dummy-project" {
			_, err := client.ProjectsApi.DeleteProject(ctx, p.Key)
			require.NoError(t, err)
		}
	}
}
