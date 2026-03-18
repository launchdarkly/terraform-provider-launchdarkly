package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func testAccFlagTemplatesConfig(projectKey string) string {
	return fmt.Sprintf(`
resource "launchdarkly_project" "test" {
	lifecycle {
		ignore_changes = [environments]
	}
	name = "Flag Templates Test Project"
	key  = "%s"
	environments {
		name  = "testEnvironment"
		key   = "test"
		color = "000000"
	}
}

resource "launchdarkly_flag_templates" "test" {
	project_key = launchdarkly_project.test.key

	tags      = ["terraform"]
	temporary = false

	boolean_defaults {
		true_display_name  = "True"
		false_display_name = "False"
		true_description   = ""
		false_description  = ""
		on_variation       = 0
		off_variation      = 1
	}
}
`, projectKey)
}

func testAccFlagTemplatesConfigUpdate(projectKey string) string {
	return fmt.Sprintf(`
resource "launchdarkly_project" "test" {
	lifecycle {
		ignore_changes = [environments]
	}
	name = "Flag Templates Test Project"
	key  = "%s"
	environments {
		name  = "testEnvironment"
		key   = "test"
		color = "000000"
	}
}

resource "launchdarkly_flag_templates" "test" {
	project_key = launchdarkly_project.test.key

	tags      = ["terraform", "updated"]
	temporary = true

	boolean_defaults {
		true_display_name  = "Enabled"
		false_display_name = "Disabled"
		true_description   = "Flag is enabled"
		false_description  = "Flag is disabled"
		on_variation       = 0
		off_variation      = 1
	}
}
`, projectKey)
}

func TestAccFlagTemplates_CreateAndUpdate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_flag_templates.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccFlagTemplatesConfig(projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlagTemplatesExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, TEMPORARY, "false"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "boolean_defaults.0.true_display_name", "True"),
					resource.TestCheckResourceAttr(resourceName, "boolean_defaults.0.false_display_name", "False"),
					resource.TestCheckResourceAttr(resourceName, "boolean_defaults.0.true_description", ""),
					resource.TestCheckResourceAttr(resourceName, "boolean_defaults.0.false_description", ""),
					resource.TestCheckResourceAttr(resourceName, "boolean_defaults.0.on_variation", "0"),
					resource.TestCheckResourceAttr(resourceName, "boolean_defaults.0.off_variation", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccFlagTemplatesConfigUpdate(projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlagTemplatesExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, TEMPORARY, "true"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "boolean_defaults.0.true_display_name", "Enabled"),
					resource.TestCheckResourceAttr(resourceName, "boolean_defaults.0.false_display_name", "Disabled"),
					resource.TestCheckResourceAttr(resourceName, "boolean_defaults.0.true_description", "Flag is enabled"),
					resource.TestCheckResourceAttr(resourceName, "boolean_defaults.0.false_description", "Flag is disabled"),
					resource.TestCheckResourceAttr(resourceName, "boolean_defaults.0.on_variation", "0"),
					resource.TestCheckResourceAttr(resourceName, "boolean_defaults.0.off_variation", "1"),
				),
			},
		},
	})
}

func testAccCheckFlagTemplatesExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("flag templates ID is not set")
		}
		client := testAccProvider.Meta().(*Client)
		_, _, err := client.ld.ProjectsApi.GetFlagDefaultsByProject(client.ctx, rs.Primary.ID).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting flag templates: %s", err)
		}
		return nil
	}
}
