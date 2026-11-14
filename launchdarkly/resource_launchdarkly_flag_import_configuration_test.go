package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// NOTE FOR REVIEWERS: these tests exercise a beta integration resource. The
// `integration_key` and the keys inside `config` must match the chosen
// integration's manifest `formVariables`. The values below target `split` and
// use the keys the `split` integration requires (`workspaceApiKey`,
// `workspaceId`, `environmentId`, `ldApiKey`). The flag-import API stores these
// values without validating the supplied credentials against the external
// system at create time, so placeholder values are sufficient for CI.
const testAccFlagImportConfigurationCreate = `
resource "launchdarkly_flag_import_configuration" "test" {
	project_key     = launchdarkly_project.test.key
	integration_key = "split"
	name            = "terraform flag import test"

	config = jsonencode({
		workspaceApiKey = "placeholder-admin-key"
		workspaceId     = "placeholder-workspace-id"
		environmentId   = "placeholder-environment-id"
		ldApiKey        = "placeholder-ld-api-key"
	})

	tags = ["terraform", "import"]
}
`

const testAccFlagImportConfigurationUpdate = `
resource "launchdarkly_flag_import_configuration" "test" {
	project_key     = launchdarkly_project.test.key
	integration_key = "split"
	name            = "terraform flag import test updated"

	config = jsonencode({
		workspaceApiKey = "placeholder-admin-key"
		workspaceId     = "placeholder-workspace-id"
		environmentId   = "placeholder-environment-id-staging"
		ldApiKey        = "placeholder-ld-api-key"
	})

	tags = ["terraform"]
}
`

func TestAccFlagImportConfiguration_CreateUpdate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_flag_import_configuration.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckFlagImportConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFlagImportConfigurationCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFlagImportConfigurationExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, ID),
					resource.TestCheckResourceAttrSet(resourceName, INTEGRATION_ID),
					resource.TestCheckResourceAttr(resourceName, INTEGRATION_KEY, "split"),
					resource.TestCheckResourceAttr(resourceName, NAME, "terraform flag import test"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// `config` holds a secret that the API masks on read, so the
				// imported value will not match the configured value. `version`
				// is a server-managed counter that the backend increments
				// asynchronously after create (the import config schedules
				// background work), so the value read at import time can differ
				// from the one captured at create time. It is computed-only, so
				// this drift never surfaces as a plan diff.
				ImportStateVerifyIgnore: []string{CONFIG, VERSION},
			},
			{
				Config: withRandomProject(projectKey, testAccFlagImportConfigurationUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlagImportConfigurationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "terraform flag import test updated"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
				),
			},
		},
	})
}

func testAccCheckFlagImportConfigurationExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		projKey, ok := rs.Primary.Attributes[PROJECT_KEY]
		if !ok {
			return fmt.Errorf("project key not found: %s", resourceName)
		}
		integrationKey := rs.Primary.Attributes[INTEGRATION_KEY]
		integrationID := rs.Primary.Attributes[INTEGRATION_ID]
		beta, err := newFlagImportConfigurationBetaClient(mustTestAccClient())
		if err != nil {
			return err
		}
		_, _, err = beta.ld.FlagImportConfigurationsBetaApi.GetFlagImportConfiguration(beta.ctx, projKey, integrationKey, integrationID).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting flag import configuration: %s", err)
		}
		return nil
	}
}

func testAccCheckFlagImportConfigurationDestroy(s *terraform.State) error {
	beta, err := newFlagImportConfigurationBetaClient(mustTestAccClient())
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_flag_import_configuration" {
			continue
		}
		projKey := rs.Primary.Attributes[PROJECT_KEY]
		integrationKey := rs.Primary.Attributes[INTEGRATION_KEY]
		integrationID := rs.Primary.Attributes[INTEGRATION_ID]
		_, res, err := beta.ld.FlagImportConfigurationsBetaApi.GetFlagImportConfiguration(beta.ctx, projKey, integrationKey, integrationID).Execute()
		if isStatusNotFound(res) {
			continue
		}
		if err != nil {
			return fmt.Errorf("unexpected error checking flag import configuration %q destruction in project %q: %s", integrationID, projKey, handleLdapiErr(err))
		}
		return fmt.Errorf("flag import configuration %q still exists in project %q", integrationID, projKey)
	}
	return nil
}
