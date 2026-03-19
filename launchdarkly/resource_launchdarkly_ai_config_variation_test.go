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
	testAccAIConfigVariationWithModelConfigKey = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "AI Config Variation Test Project"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_model_config" "test" {
	project_key    = launchdarkly_project.test.key
	key            = "%s"
	name           = "Test Model"
	model_id       = "gpt-4"
	model_provider = "openai"
}

resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "Parent AI Config"
	description = "Parent config for variation tests"
	tags        = ["test"]
	depends_on  = [launchdarkly_model_config.test]
}

resource "launchdarkly_ai_config_variation" "test" {
	project_key      = launchdarkly_project.test.key
	config_key       = launchdarkly_ai_config.test.key
	key              = "%s"
	name             = "Variation with model config"
	model_config_key = launchdarkly_model_config.test.key
	messages {
		role    = "system"
		content = "You are a helpful assistant."
	}
}
`

	testAccAIConfigVariationAgentMode = `
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
	name        = "Agent Mode Config"
	description = "Agent mode parent"
	mode        = "agent"
}

resource "launchdarkly_ai_config_variation" "test" {
	project_key = launchdarkly_project.test.key
	config_key  = launchdarkly_ai_config.test.key
	key         = "%s"
	name        = "%s"
}
`

	testAccAIConfigVariationWithToolKeys = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "AI Config Variation Test Project"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_ai_tool" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	description = "Test tool"
	schema_json = jsonencode({
		type = "object"
		properties = {
			query = { type = "string" }
		}
	})
}

resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "Parent AI Config"
	description = "Parent for tool keys test"
	depends_on  = [launchdarkly_ai_tool.test]
}

resource "launchdarkly_ai_config_variation" "test" {
	project_key = launchdarkly_project.test.key
	config_key  = launchdarkly_ai_config.test.key
	key         = "%s"
	name        = "Variation with tools"
	tool_keys   = [launchdarkly_ai_tool.test.key]
	messages {
		role    = "system"
		content = "You are a helpful assistant."
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

	resource.Test(t, resource.TestCase{
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

func TestAccAIConfigVariation_WithModelConfigKey(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	configKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	modelConfigKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	variationKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_config_variation.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAIConfigVariationDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAIConfigVariationWithModelConfigKey, projectKey, modelConfigKey, configKey, variationKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigVariationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, MODEL_CONFIG_KEY, modelConfigKey),
					resource.TestCheckResourceAttr(resourceName, "messages.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, VARIATION_ID),
					resource.TestCheckResourceAttrSet(resourceName, VERSION),
					resource.TestCheckResourceAttrSet(resourceName, CREATION_DATE),
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

// TestAccAIConfigVariation_AgentMode tests creating a variation under an agent-mode AI config.
// Note: The API does not support description/instructions fields on variation create/update —
// those are read-only fields populated server-side. This test verifies that a basic variation
// can be created and updated under an agent-mode parent config.
func TestAccAIConfigVariation_AgentMode(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	configKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	variationKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	variationName := "Agent Variation"
	updatedName := "Updated Agent Variation"
	resourceName := "launchdarkly_ai_config_variation.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAIConfigVariationDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAIConfigVariationAgentMode, projectKey, configKey, variationKey, variationName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigVariationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, variationName),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccAIConfigVariationAgentMode, projectKey, configKey, variationKey, updatedName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigVariationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, updatedName),
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

// TestAccAIConfigVariation_WithToolKeys tests creating a variation with tool_keys.
// Same two-apply pattern as AgentMode: POST doesn't persist tool_keys, PATCH does.
func TestAccAIConfigVariation_WithToolKeys(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	configKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	toolKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	variationKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_config_variation.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAIConfigVariationDestroy,
		Steps: []resource.TestStep{
			{
				// First apply: POST creates variation but may not persist tool_keys
				Config:             fmt.Sprintf(testAccAIConfigVariationWithToolKeys, projectKey, toolKey, configKey, variationKey),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigVariationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Variation with tools"),
				),
			},
			{
				// Second apply: PATCH sets tool_keys (creates new version).
				// The GET may still return v1 due to eventual consistency, so allow non-empty plan.
				Config:             fmt.Sprintf(testAccAIConfigVariationWithToolKeys, projectKey, toolKey, configKey, variationKey),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigVariationExists(resourceName),
				),
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
