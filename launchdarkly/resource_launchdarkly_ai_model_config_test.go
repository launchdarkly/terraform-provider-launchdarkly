package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	testAccAIModelConfigBasic = `
resource "launchdarkly_ai_model_config" "basic" {
	project_key = launchdarkly_project.test.key
	key         = "basic-ai-model"
	name        = "Basic AI Model"
	model_id    = "gpt-4"
}
`
	testAccAIModelConfigFull = `
resource "launchdarkly_ai_model_config" "full" {
	project_key          = launchdarkly_project.test.key
	key                  = "full-ai-model"
	name                 = "Full AI Model"
	model_id             = "gpt-4-turbo"
	model_provider       = "openai"
	icon                 = "openai-icon"
	tags                 = ["test", "ai"]
	cost_per_input_token  = 0.00001
	cost_per_output_token = 0.00003
	params = {
		temperature = "0.7"
		max_tokens  = "1000"
	}
	custom_params = {
		custom_key = "custom_value"
	}
}
`
)

func TestAccAIModelConfig_BasicCreate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_model_config.basic"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccAIModelConfigBasic),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckAIModelConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "basic-ai-model"),
					resource.TestCheckResourceAttr(resourceName, NAME, "Basic AI Model"),
					resource.TestCheckResourceAttr(resourceName, MODEL_ID, "gpt-4"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttrSet(resourceName, VERSION),
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

func TestAccAIModelConfig_FullCreate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_model_config.full"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccAIModelConfigFull),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckAIModelConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "full-ai-model"),
					resource.TestCheckResourceAttr(resourceName, NAME, "Full AI Model"),
					resource.TestCheckResourceAttr(resourceName, MODEL_ID, "gpt-4-turbo"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, MODEL_PROVIDER, "openai"),
					resource.TestCheckResourceAttr(resourceName, ICON, "openai-icon"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, COST_PER_INPUT_TOKEN, "0.00001"),
					resource.TestCheckResourceAttr(resourceName, COST_PER_OUTPUT_TOKEN, "0.00003"),
					resource.TestCheckResourceAttr(resourceName, "params.temperature", "0.7"),
					resource.TestCheckResourceAttr(resourceName, "params.max_tokens", "1000"),
					resource.TestCheckResourceAttr(resourceName, "custom_params.custom_key", "custom_value"),
					resource.TestCheckResourceAttrSet(resourceName, VERSION),
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

func TestAccAIModelConfig_ForceNewOnKeyChange(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_model_config.basic"

	config1 := `
resource "launchdarkly_ai_model_config" "basic" {
	project_key = launchdarkly_project.test.key
	key         = "ai-model-v1"
	name        = "AI Model V1"
	model_id    = "gpt-4"
}
`
	config2 := `
resource "launchdarkly_ai_model_config" "basic" {
	project_key = launchdarkly_project.test.key
	key         = "ai-model-v2"
	name        = "AI Model V2"
	model_id    = "gpt-4"
}
`

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, config1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckAIModelConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "ai-model-v1"),
					resource.TestCheckResourceAttr(resourceName, NAME, "AI Model V1"),
				),
			},
			{
				Config: withRandomProject(projectKey, config2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckAIModelConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "ai-model-v2"),
					resource.TestCheckResourceAttr(resourceName, NAME, "AI Model V2"),
				),
			},
		},
	})
}

func testAccCheckAIModelConfigExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		aiModelConfigKey, ok := rs.Primary.Attributes[KEY]
		if !ok {
			return fmt.Errorf("AI model config key not found: %s", resourceName)
		}
		projKey, ok := rs.Primary.Attributes[PROJECT_KEY]
		if !ok {
			return fmt.Errorf("project key not found: %s", resourceName)
		}
		client := testAccProvider.Meta().(*Client)
		_, _, err := client.ldBeta.AIConfigsBetaApi.GetModelConfig(client.ctx, projKey, aiModelConfigKey).LDAPIVersion("beta").Execute()
		if err != nil {
			return fmt.Errorf("received an error getting AI model config. %s", err)
		}
		return nil
	}
}


