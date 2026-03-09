package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	testAccAiConfigBasic = `
resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "test-ai-config"
	name        = "Test AI Config"
	description = "A test AI Config."
	tags        = ["test"]
}
`
	testAccAiConfigUpdate = `
resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "test-ai-config"
	name        = "Updated AI Config"
	description = "An updated test AI Config."
	tags        = ["test", "updated"]
}
`
	testAccAiConfigWithMode = `
resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "test-ai-config-mode"
	name        = "AI Config With Mode"
	description = "An AI Config with mode set."
	mode        = "completion"
	tags        = ["test"]
}
`
	testAccAiConfigWithOptionalFields = `
resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "test-ai-config-optional"
	name        = "AI Config With Optionals"
	description = "A test AI Config with optional fields."
	tags        = ["test", "optional"]
}
`
	testAccAiConfigWithoutOptionalFields = `
resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "test-ai-config-optional"
	name        = "AI Config With Optionals"
}
`
)

func TestAccAiConfig_BasicCreateAndUpdate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_config.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAiConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccAiConfigBasic),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckAiConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Test AI Config"),
					resource.TestCheckResourceAttr(resourceName, KEY, "test-ai-config"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "A test AI Config."),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withRandomProject(projectKey, testAccAiConfigUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckAiConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Updated AI Config"),
					resource.TestCheckResourceAttr(resourceName, KEY, "test-ai-config"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "An updated test AI Config."),
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

func TestAccAiConfig_WithMode(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_config.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAiConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccAiConfigWithMode),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckAiConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "AI Config With Mode"),
					resource.TestCheckResourceAttr(resourceName, KEY, "test-ai-config-mode"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, MODE, "completion"),
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

func TestAccAiConfig_RemoveOptionalFields(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_config.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAiConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccAiConfigWithOptionalFields),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckAiConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "AI Config With Optionals"),
					resource.TestCheckResourceAttr(resourceName, KEY, "test-ai-config-optional"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "A test AI Config with optional fields."),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withRandomProject(projectKey, testAccAiConfigWithoutOptionalFields),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAiConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "AI Config With Optionals"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, ""),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "0"),
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

func testAccCheckAiConfigExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("AI Config ID is not set")
		}
		configKey := rs.Primary.Attributes[KEY]
		projectKey := rs.Primary.Attributes[PROJECT_KEY]

		client := testAccProvider.Meta().(*Client)

		_, _, err := getAiConfig(client, projectKey, configKey)
		if err != nil {
			return fmt.Errorf("received an error getting AI Config: %s", err)
		}
		return nil
	}
}

func testAccCheckAiConfigDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_ai_config" {
			continue
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		configKey := rs.Primary.Attributes[KEY]

		_, resp, err := getAiConfig(client, projectKey, configKey)
		if isStatusNotFound(resp) {
			continue
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("AI Config %s/%s still exists", projectKey, configKey)
	}
	return nil
}
