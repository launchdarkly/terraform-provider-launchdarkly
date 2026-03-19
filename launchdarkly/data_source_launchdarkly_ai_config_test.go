package launchdarkly

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceAiConfig_Basic(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "data.launchdarkly_ai_config.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, `
resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "test-ai-config-ds"
	name        = "Test AI Config DS"
	description = "A test AI Config for data source."
	tags        = ["test"]
}

data "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = launchdarkly_ai_config.test.key
}
`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAiConfigExists("launchdarkly_ai_config.test"),
					resource.TestCheckResourceAttr(resourceName, NAME, "Test AI Config DS"),
					resource.TestCheckResourceAttr(resourceName, KEY, "test-ai-config-ds"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "A test AI Config for data source."),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
				),
			},
		},
	})
}
