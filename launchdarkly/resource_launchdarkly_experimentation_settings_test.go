package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccExperimentationSettingsBasic = `
resource "launchdarkly_context_kind" "organization" {
	project_key = launchdarkly_project.test.key
	key         = "organization"
	name        = "Organization"
}

resource "launchdarkly_experimentation_settings" "test" {
	project_key = launchdarkly_project.test.key

	randomization_units = [
		{
			randomization_unit = "user"
			default            = true
		},
		{
			randomization_unit = launchdarkly_context_kind.organization.key
		},
	]
}
`

const testAccExperimentationSettingsUpdate = `
resource "launchdarkly_context_kind" "organization" {
	project_key = launchdarkly_project.test.key
	key         = "organization"
	name        = "Organization"
}

resource "launchdarkly_experimentation_settings" "test" {
	project_key = launchdarkly_project.test.key

	randomization_units = [
		{
			randomization_unit = "user"
		},
		{
			randomization_unit = launchdarkly_context_kind.organization.key
			default            = true
		},
	]
}
`

func TestAccExperimentationSettings_CreateUpdate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_experimentation_settings.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccExperimentationSettingsBasic),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ID, projectKey),
					resource.TestCheckResourceAttr(resourceName, "randomization_units.0.randomization_unit", "user"),
					resource.TestCheckResourceAttr(resourceName, "randomization_units.0.default", "true"),
					resource.TestCheckResourceAttr(resourceName, "randomization_units.1.randomization_unit", "organization"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withRandomProject(projectKey, testAccExperimentationSettingsUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "randomization_units.1.randomization_unit", "organization"),
					resource.TestCheckResourceAttr(resourceName, "randomization_units.1.default", "true"),
				),
			},
		},
	})
}

func TestAccDataSourceExperimentationSettings_basic(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	dataSourceName := "data.launchdarkly_experimentation_settings.test"

	config := withRandomProject(projectKey, fmt.Sprintf(`%s

data "launchdarkly_experimentation_settings" "test" {
	project_key = launchdarkly_experimentation_settings.test.project_key
}
`, testAccExperimentationSettingsBasic))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(dataSourceName, ID, projectKey),
					resource.TestCheckResourceAttrSet(dataSourceName, "randomization_units.0.randomization_unit"),
				),
			},
		},
	})
}
