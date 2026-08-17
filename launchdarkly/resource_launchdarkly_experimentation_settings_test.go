package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// The experimentation settings resource is a per-project singleton. Each
// randomization unit must correspond to an existing context kind, so the
// update step creates an `account` context kind before referencing it.

const testAccExperimentationSettingsBasic = `
resource "launchdarkly_experimentation_settings" "test" {
	project_key = launchdarkly_project.test.key
	randomization_units = {
		user = {
			default = true
		}
	}
}
`

const testAccExperimentationSettingsUpdate = `
resource "launchdarkly_context_kind" "account" {
	project_key = launchdarkly_project.test.key
	key         = "account"
	name        = "Account"
}

resource "launchdarkly_experimentation_settings" "test" {
	project_key = launchdarkly_project.test.key
	randomization_units = {
		user = {
			default = true
		}
		account = {
			default = false
		}
	}

	depends_on = [launchdarkly_context_kind.account]
}
`

func TestAccExperimentationSettings_CreateAndUpdate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_experimentation_settings.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckExperimentationSettingsDestroy,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccExperimentationSettingsBasic),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckExperimentationSettingsExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ID, projectKey),
					resource.TestCheckResourceAttr(resourceName, "randomization_units.user.default", "true"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update step ADDS a new randomization unit entry to the map.
			{
				Config: withRandomProject(projectKey, testAccExperimentationSettingsUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckExperimentationSettingsExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, "randomization_units.user.default", "true"),
					resource.TestCheckResourceAttr(resourceName, "randomization_units.account.default", "false"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckExperimentationSettingsExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		projKey, ok := rs.Primary.Attributes[PROJECT_KEY]
		if !ok {
			return fmt.Errorf("project key not found: %s", resourceName)
		}
		client := mustTestAccClient()
		_, _, err := client.ld.ExperimentsApi.GetExperimentationSettings(client.ctx, projKey).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting experimentation settings: %s", err)
		}
		return nil
	}
}

// testAccCheckExperimentationSettingsDestroy verifies that the settings are no
// longer reachable after destroy. The resource has no delete endpoint, but the
// enclosing project is destroyed, so the settings GET returns a 404.
func testAccCheckExperimentationSettingsDestroy(s *terraform.State) error {
	client := mustTestAccClient()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_experimentation_settings" {
			continue
		}
		projKey := rs.Primary.Attributes[PROJECT_KEY]
		_, res, err := client.ld.ExperimentsApi.GetExperimentationSettings(client.ctx, projKey).Execute()
		if isStatusNotFound(res) {
			continue
		}
		if err != nil {
			return fmt.Errorf("unexpected error checking experimentation settings destroy for project %q: %s", projKey, err)
		}
		return fmt.Errorf("experimentation settings for project %q still exist", projKey)
	}
	return nil
}
