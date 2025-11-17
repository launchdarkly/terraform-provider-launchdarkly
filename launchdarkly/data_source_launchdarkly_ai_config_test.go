package launchdarkly

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	testAccDataSourceAIConfig = `
resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "test-ai-config"
	name        = "Test AI Config"
	description = "Test description"
	tags        = ["terraform", "test"]
}

data "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_ai_config.test.project_key
	key         = launchdarkly_ai_config.test.key
}
`

	testAccDataSourceAIConfigWithTeamFmt = `
resource "launchdarkly_team" "test" {
	key  = "%s"
	name = "Test Team"
}

resource "launchdarkly_ai_config" "test" {
	project_key         = launchdarkly_project.test.key
	key                 = "test-ai-config"
	name                = "Test AI Config"
	maintainer_team_key = launchdarkly_team.test.key
}

data "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_ai_config.test.project_key
	key         = launchdarkly_ai_config.test.key
}
`
)

func TestAccDataSourceAIConfig_exists(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "data.launchdarkly_ai_config.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccDataSourceAIConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, ID),
					resource.TestCheckResourceAttr(resourceName, NAME, "Test AI Config"),
					resource.TestCheckResourceAttr(resourceName, KEY, "test-ai-config"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "Test description"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttrSet(resourceName, VERSION),
				),
			},
		},
	})
}

func TestAccDataSourceAIConfig_existsWithTeamMaintainer(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	teamKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "data.launchdarkly_ai_config.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, fmt.Sprintf(testAccDataSourceAIConfigWithTeamFmt, teamKey)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, ID),
					resource.TestCheckResourceAttr(resourceName, NAME, "Test AI Config"),
					resource.TestCheckResourceAttr(resourceName, KEY, "test-ai-config"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, MAINTAINER_TEAM_KEY, teamKey),
					resource.TestCheckResourceAttrSet(resourceName, VERSION),
				),
			},
		},
	})
}

func TestAccDataSourceAIConfig_noMatchReturnsError(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	aiConfigKey := acctest.RandStringFromCharSet(24, acctest.CharSetAlphaNum)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, fmt.Sprintf(`
data "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
}
`, aiConfigKey)),
				ExpectError: regexp.MustCompile(`failed to get AI config`),
			},
		},
	})
}
