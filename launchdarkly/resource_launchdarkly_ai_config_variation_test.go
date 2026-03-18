package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	testAccAIConfigVariationCreate = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "AI Config Variation Test Project"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "Parent AI Config"
	description = "Parent config for variation tests"
	tags        = ["test"]
}

resource "launchdarkly_ai_config_variation" "test" {
	project_key = launchdarkly_project.test.key
	config_key  = launchdarkly_ai_config.test.key
	key         = "%s"
	name        = "%s"
	messages {
		role    = "system"
		content = "You are a helpful assistant."
	}
}
`

	testAccAIConfigVariationUpdate = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "AI Config Variation Test Project"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "Parent AI Config"
	description = "Parent config for variation tests"
	tags        = ["test"]
}

resource "launchdarkly_ai_config_variation" "test" {
	project_key = launchdarkly_project.test.key
	config_key  = launchdarkly_ai_config.test.key
	key         = "%s"
	name        = "%s"
	messages {
		role    = "system"
		content = "You are an expert assistant."
	}
	messages {
		role    = "user"
		content = "Hello!"
	}
}
`
)

func TestAccAIConfigVariation_CreateAndUpdate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	configKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	variationKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	variationName := "Test Variation"
	updatedVariationName := "Updated Variation"
	resourceName := "launchdarkly_ai_config_variation.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAIConfigVariationDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAIConfigVariationCreate, projectKey, configKey, variationKey, variationName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigVariationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, AI_CONFIG_KEY, configKey),
					resource.TestCheckResourceAttr(resourceName, KEY, variationKey),
					resource.TestCheckResourceAttr(resourceName, NAME, variationName),
					resource.TestCheckResourceAttr(resourceName, "messages.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "messages.0.role", "system"),
					resource.TestCheckResourceAttr(resourceName, "messages.0.content", "You are a helpful assistant."),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccAIConfigVariationUpdate, projectKey, configKey, variationKey, updatedVariationName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigVariationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, updatedVariationName),
					resource.TestCheckResourceAttr(resourceName, "messages.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "messages.0.role", "system"),
					resource.TestCheckResourceAttr(resourceName, "messages.0.content", "You are an expert assistant."),
					resource.TestCheckResourceAttr(resourceName, "messages.1.role", "user"),
					resource.TestCheckResourceAttr(resourceName, "messages.1.content", "Hello!"),
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

func testAccCheckAIConfigVariationExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("AI config variation ID is not set")
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		configKey := rs.Primary.Attributes[AI_CONFIG_KEY]
		variationKey := rs.Primary.Attributes[KEY]

		client := testAccProvider.Meta().(*Client)
		_, _, err := client.ld.AIConfigsApi.GetAIConfigVariation(client.ctx, projectKey, configKey, variationKey).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting AI config variation: %s", err)
		}
		return nil
	}
}

var testAccCheckAIConfigVariationDestroy = func(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_ai_config_variation" {
			continue
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		configKey := rs.Primary.Attributes[AI_CONFIG_KEY]
		variationKey := rs.Primary.Attributes[KEY]

		_, res, err := client.ld.AIConfigsApi.GetAIConfigVariation(client.ctx, projectKey, configKey, variationKey).Execute()
		if isStatusNotFound(res) {
			continue
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("AI config variation %s/%s/%s still exists", projectKey, configKey, variationKey)
	}
	return nil
}
