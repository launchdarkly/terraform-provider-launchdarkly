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

	testAccAIConfigVariationWithInlineModel = `
resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "Parent AI Config"
	description = "Parent for inline model test"
	tags        = ["test"]
}

resource "launchdarkly_ai_config_variation" "test" {
	project_key = launchdarkly_project.test.key
	config_key  = launchdarkly_ai_config.test.key
	key         = "%s"
	name        = "Variation with inline model"
	model       = jsonencode({
		modelName  = "gpt-4"
		parameters = { temperature = 0.7 }
	})
	messages {
		role    = "system"
		content = "You are a helpful assistant."
	}
}
`

	testAccAIConfigVariationWithToolKeys = `
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
	aiTestCooldown()
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
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigVariationCreate, configKey, variationKey, variationName)),
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
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigVariationUpdate, configKey, variationKey, updatedVariationName)),
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
	aiTestCooldown()
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
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigVariationWithModelConfigKey, modelConfigKey, configKey, variationKey)),
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
// Note: The API does not support description/instructions fields on variation create/update --
// those are read-only fields populated server-side. This test verifies that a basic variation
// can be created and updated under an agent-mode parent config.
func TestAccAIConfigVariation_AgentMode(t *testing.T) {
	aiTestCooldown()
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
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigVariationAgentMode, configKey, variationKey, variationName)),
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
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigVariationAgentMode, configKey, variationKey, updatedName)),
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
func TestAccAIConfigVariation_WithToolKeys(t *testing.T) {
	aiTestCooldown()
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
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigVariationWithToolKeys, toolKey, configKey, variationKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigVariationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Variation with tools"),
					resource.TestCheckResourceAttr(resourceName, "tool_keys.#", "1"),
				),
				// The API GET response does not return the `tools` field, so tool_keys
				// cannot be read back from the API. This causes a persistent diff on refresh.
				// TODO: remove once the API returns tools in the GET variation response.
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAIConfigVariation_WithInlineModel(t *testing.T) {
	aiTestCooldown()
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	configKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	variationKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_config_variation.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAIConfigVariationDestroy,
		Steps: []resource.TestStep{
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigVariationWithInlineModel, configKey, variationKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigVariationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Variation with inline model"),
					resource.TestCheckResourceAttrSet(resourceName, MODEL),
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
