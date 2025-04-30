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
	key = "%s"
	name = "test project"
}

resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key = "%s"
	name = "Test AI Config"
	description = "This is a test AI Config"
	tags = ["test", "terraform"]
	
	variations {
		key = "variation-1"
		name = "Variation 1"
		description = "First variation"
		model = "gpt-4"
		parameters = {
			"temperature" = "0.7"
			"max_tokens" = "1000"
		}
	}
	
	variations {
		key = "variation-2"
		name = "Variation 2"
		description = "Second variation"
		model = "gpt-3.5-turbo"
		parameters = {
			"temperature" = "0.5"
			"max_tokens" = "500"
		}
	}
}
`

	testAccAIConfigUpdate = `
resource "launchdarkly_project" "test" {
	key = "%s"
	name = "test project"
}

resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key = "%s"
	name = "Updated AI Config"
	description = "This is an updated test AI Config"
	tags = ["test", "terraform", "updated"]
	
	variations {
		key = "variation-1"
		name = "Updated Variation 1"
		description = "Updated first variation"
		model = "gpt-4"
		parameters = {
			"temperature" = "0.8"
			"max_tokens" = "1500"
		}
	}
	
	variations {
		key = "variation-2"
		name = "Updated Variation 2"
		description = "Updated second variation"
		model = "gpt-3.5-turbo"
		parameters = {
			"temperature" = "0.6"
			"max_tokens" = "800"
		}
	}
	
	variations {
		key = "variation-3"
		name = "Variation 3"
		description = "Third variation"
		model = "claude-2"
		parameters = {
			"temperature" = "0.4"
			"max_tokens" = "2000"
		}
	}
}
`
)

func TestAccAIConfig_Basic(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	configKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_config.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAIConfigCreate, projectKey, configKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", configKey),
					resource.TestCheckResourceAttr(resourceName, "name", "Test AI Config"),
					resource.TestCheckResourceAttr(resourceName, "description", "This is a test AI Config"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.key", "variation-1"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.name", "Variation 1"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.model", "gpt-4"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.parameters.temperature", "0.7"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.parameters.max_tokens", "1000"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccAIConfigUpdate, projectKey, configKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", configKey),
					resource.TestCheckResourceAttr(resourceName, "name", "Updated AI Config"),
					resource.TestCheckResourceAttr(resourceName, "description", "This is an updated test AI Config"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.key", "variation-1"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.name", "Updated Variation 1"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.parameters.temperature", "0.8"),
					resource.TestCheckResourceAttr(resourceName, "variations.2.key", "variation-3"),
					resource.TestCheckResourceAttr(resourceName, "variations.2.model", "claude-2"),
				),
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
			return fmt.Errorf("no ID is set")
		}

		client := testAccProvider.Meta().(*Client)
		projectKey, configKey, err := aiConfigIdToKeys(rs.Primary.ID)
		if err != nil {
			return err
		}

		_, _, err = client.getAIConfig(projectKey, configKey)
		if err != nil {
			return fmt.Errorf("received an error retrieving ai config %q: %s", rs.Primary.ID, err)
		}
		return nil
	}
}
