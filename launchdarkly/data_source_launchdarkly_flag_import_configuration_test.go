package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go/v23"
	"github.com/stretchr/testify/require"
)

const testAccDataSourceFlagImportConfiguration = `
data "launchdarkly_flag_import_configuration" "testing" {
	project_key     = "%s"
	integration_key = "%s"
	integration_id  = "%s"
}
`

// testAccDataSourceFlagImportConfigurationScaffold creates a project and a flag
// import configuration via the API (the endpoints are beta, so the
// configuration is created with the beta client) so the data source can read it
// back. NOTE FOR REVIEWERS: the `integration_key` and `config` keys must match
// the chosen integration's manifest; `split` is used here with the keys that
// integration requires. The flag-import API stores the config without
// validating the credentials at create time, so placeholders are sufficient.
func testAccDataSourceFlagImportConfigurationScaffold(client *Client, beta *Client, projectKey string) (*ldapi.FlagImportIntegration, error) {
	projectBody := ldapi.ProjectPost{
		Name: "Flag Import Config Test Project",
		Key:  projectKey,
	}
	project, err := testAccProjectScaffoldCreate(client, projectBody)
	if err != nil {
		return nil, err
	}

	name := "flag import ds test"
	post := ldapi.FlagImportConfigurationPost{
		Config: map[string]interface{}{
			"workspaceApiKey": "placeholder-admin-key",
			"workspaceId":     "placeholder-workspace-id",
			"environmentId":   "placeholder-environment-id",
			"ldApiKey":        "placeholder-ld-api-key",
		},
		Tags: []string{"test"},
		Name: &name,
	}
	cfg, _, err := beta.ld.FlagImportConfigurationsBetaApi.CreateFlagImportConfiguration(beta.ctx, project.Key, "split").FlagImportConfigurationPost(post).Execute()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func TestAccDataSourceFlagImportConfiguration_noMatchReturnsError(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectBody := ldapi.ProjectPost{
		Name: "Terraform Flag Import Config Test Project",
		Key:  projectKey,
	}
	project, err := testAccProjectScaffoldCreate(client, projectBody)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, testAccProjectScaffoldDelete(client, projectKey))
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// A well-formed but nonexistent integration id (UUID shape).
				// A non-UUID value would instead return a 400 "malformed
				// integration id" from the API before the not-found check.
				Config:      fmt.Sprintf(testAccDataSourceFlagImportConfiguration, project.Key, "split", "00000000-0000-0000-0000-000000000000"),
				ExpectError: regexp.MustCompile("Error: 404 Not Found"),
			},
		},
	})
}

func TestAccDataSourceFlagImportConfiguration_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)
	beta, err := newFlagImportConfigurationBetaClient(client)
	require.NoError(t, err)

	cfg, err := testAccDataSourceFlagImportConfigurationScaffold(client, beta, projectKey)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, testAccProjectScaffoldDelete(client, projectKey))
	}()

	resourceName := "data.launchdarkly_flag_import_configuration.testing"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceFlagImportConfiguration, projectKey, cfg.GetIntegrationKey(), cfg.GetId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, INTEGRATION_ID),
					resource.TestCheckResourceAttr(resourceName, INTEGRATION_KEY, cfg.GetIntegrationKey()),
					resource.TestCheckResourceAttr(resourceName, NAME, cfg.GetName()),
					resource.TestCheckResourceAttr(resourceName, ID, projectKey+"/"+cfg.GetIntegrationKey()+"/"+cfg.GetId()),
				),
			},
		},
	})
}
