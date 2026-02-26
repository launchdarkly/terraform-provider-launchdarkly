package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	testAccAIConfigBasic = `
resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "test-ai-config"
	name        = "Test AI Config"
}
`

	testAccAIConfigUpdate = `
resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "test-ai-config"
	name        = "Updated AI Config"
	description = "Updated description"
	tags        = ["terraform", "updated"]
}
`

	testAccAIConfigWithTags = `
resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "test-ai-config"
	name        = "Test AI Config"
	description = "AI Config with tags"
	tags        = ["terraform", "test"]
}
`
)

func TestAccAIConfig_Create(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_config.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccAIConfigBasic),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckAIConfigExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, NAME, "Test AI Config"),
					resource.TestCheckResourceAttr(resourceName, KEY, "test-ai-config"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "0"),
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

func TestAccAIConfig_Update(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_config.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccAIConfigBasic),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckAIConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Test AI Config"),
					resource.TestCheckResourceAttr(resourceName, KEY, "test-ai-config"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "0"),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccAIConfigUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckAIConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Updated AI Config"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "Updated description"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
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

func TestAccAIConfig_WithTags(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_config.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccAIConfigWithTags),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckAIConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Test AI Config"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "AI Config with tags"),
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
		client := testAccProvider.Meta().(*Client)
		projectKey, key, err := aiConfigIdToKeys(rs.Primary.ID)
		if err != nil {
			return err
		}
		_, _, err = client.ldBeta.AIConfigsBetaApi.GetAIConfig(client.ctx, projectKey, key).LDAPIVersion("beta").Execute()
		if err != nil {
			return fmt.Errorf("received an error getting AI config: %s", err)
		}
		return nil
	}
}
