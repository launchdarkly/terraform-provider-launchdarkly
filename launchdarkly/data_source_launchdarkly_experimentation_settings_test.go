package launchdarkly

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestAccDataSourceExperimentationSettings_read(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	dataSourceName := "data.launchdarkly_experimentation_settings.test"

	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)
	// Scaffolding the project's context kinds and experimentation settings is
	// done directly via the API so the data source has data to read back.
	betaClient, err := newBetaClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)
	err = scaffoldProjectWithExperimentationSettings(client, betaClient, projectKey, []string{"user", "request"})
	require.NoError(t, err)

	defer func() {
		require.NoError(t, testAccProjectScaffoldDelete(betaClient, projectKey))
	}()

	config := fmt.Sprintf(`
data "launchdarkly_experimentation_settings" "test" {
	project_key = "%s"
}
`, projectKey)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(dataSourceName, ID, projectKey),
					resource.TestCheckResourceAttr(dataSourceName, "randomization_units.user.default", "true"),
					resource.TestCheckResourceAttr(dataSourceName, "randomization_units.request.default", "false"),
				),
			},
		},
	})
}
