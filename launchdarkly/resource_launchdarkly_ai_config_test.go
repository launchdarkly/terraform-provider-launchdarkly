package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	testAccAIConfigCreate = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "AI Config Test Project"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "%s"
	description = "%s"
	tags        = ["test"]
}
`

	testAccAIConfigUpdate = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "AI Config Test Project"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "%s"
	description = "%s"
	tags        = ["test", "updated"]
}
`
)

func TestAccAIConfig_CreateAndUpdate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	configKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	configName := "Test AI Config"
	configDescription := "Test AI config description"
	updatedConfigName := "Updated Test AI Config"
	updatedConfigDescription := "Updated AI config description"
	resourceName := "launchdarkly_ai_config.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAIConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAIConfigCreate, projectKey, configKey, configName, configDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, KEY, configKey),
					resource.TestCheckResourceAttr(resourceName, NAME, configName),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, configDescription),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccAIConfigUpdate, projectKey, configKey, updatedConfigName, updatedConfigDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, updatedConfigName),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, updatedConfigDescription),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
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

func testAccCheckAIConfigExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("AI config ID is not set")
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		configKey := rs.Primary.Attributes[KEY]

		client := testAccProvider.Meta().(*Client)
		_, _, err := client.ld.AIConfigsApi.GetAIConfig(client.ctx, projectKey, configKey).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting AI config: %s", err)
		}
		return nil
	}
}

var testAccCheckAIConfigDestroy = func(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_ai_config" {
			continue
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		configKey := rs.Primary.Attributes[KEY]

		_, res, err := client.ld.AIConfigsApi.GetAIConfig(client.ctx, projectKey, configKey).Execute()
		if isStatusNotFound(res) {
			continue
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("AI config %s/%s still exists", projectKey, configKey)
	}
	return nil
}
