package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	testAccModelConfigCreate = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "Model Config Test Project"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_model_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "%s"
	model_id    = "%s"
	model_provider = "%s"
	tags        = ["test"]
	params      = jsonencode({
		temperature = 0.7
		maxTokens   = 4096
	})
}
`
)

func TestAccModelConfig_CreateAndImport(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	modelConfigKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	modelConfigName := "Test Model Config"
	modelID := "gpt-4"
	providerName := "openai"
	resourceName := "launchdarkly_model_config.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckModelConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccModelConfigCreate, projectKey, modelConfigKey, modelConfigName, modelID, providerName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckModelConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, KEY, modelConfigKey),
					resource.TestCheckResourceAttr(resourceName, NAME, modelConfigName),
					resource.TestCheckResourceAttr(resourceName, MODEL_ID, modelID),
					resource.TestCheckResourceAttr(resourceName, PROVIDER_NAME, providerName),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
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

func testAccCheckModelConfigExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("model config ID is not set")
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		modelConfigKey := rs.Primary.Attributes[KEY]

		client := testAccProvider.Meta().(*Client)
		_, _, err := client.ld.AIConfigsApi.GetModelConfig(client.ctx, projectKey, modelConfigKey).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting model config: %s", err)
		}
		return nil
	}
}

var testAccCheckModelConfigDestroy = func(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_model_config" {
			continue
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		modelConfigKey := rs.Primary.Attributes[KEY]

		_, res, err := client.ld.AIConfigsApi.GetModelConfig(client.ctx, projectKey, modelConfigKey).Execute()
		if isStatusNotFound(res) {
			continue
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("model config %s/%s still exists", projectKey, modelConfigKey)
	}
	return nil
}
