package launchdarkly

import (
	"fmt"
	"regexp"
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

	testAccAIConfigWithTeamMaintainer = `
resource "launchdarkly_team" "test" {
	key  = "test-team"
	name = "Test Team"
}

resource "launchdarkly_ai_config" "test" {
	project_key         = launchdarkly_project.test.key
	key                 = "test-ai-config"
	name                = "Test AI Config"
	maintainer_team_key = launchdarkly_team.test.key
}
`

	testAccAIConfigConflictingMaintainers = `
resource "launchdarkly_team" "test" {
	key  = "test-team"
	name = "Test Team"
}

resource "launchdarkly_ai_config" "test" {
	project_key         = launchdarkly_project.test.key
	key                 = "test-ai-config"
	name                = "Test AI Config"
	maintainer_id       = "507f1f77bcf86cd799439011"
	maintainer_team_key = launchdarkly_team.test.key
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

func TestAccAIConfig_WithTeamMaintainer(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_config.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccAIConfigWithTeamMaintainer),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckAIConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Test AI Config"),
					resource.TestCheckResourceAttr(resourceName, MAINTAINER_TEAM_KEY, "test-team"),
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

func TestAccAIConfig_ConflictingMaintainers(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      withRandomProject(projectKey, testAccAIConfigConflictingMaintainers),
				ExpectError: regexp.MustCompile(`"maintainer_id".*conflicts with maintainer_team_key`),
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
		_, _, err = client.ld.AIConfigsBetaApi.GetAIConfig(client.ctx, projectKey, key).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting AI config: %s", err)
		}
		return nil
	}
}
