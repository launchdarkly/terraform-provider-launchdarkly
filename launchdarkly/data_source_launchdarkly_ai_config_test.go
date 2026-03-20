package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	testAccDataSourceAIConfigBasic = `
data "launchdarkly_ai_config" "test" {
	project_key = "%s"
	key         = "%s"
}
`
)

func TestAccDataSourceAIConfig_noMatchReturnsError(t *testing.T) {
	projectKey := "nonexistent-project-key"
	configKey := "nonexistent-config-key"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAIConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccDataSourceAIConfigBasic, projectKey, configKey),
				ExpectError: regexp.MustCompile(`failed to get AI config with key "nonexistent-config-key" in project "nonexistent-project-key"`),
			},
		},
	})
}

func TestAccDataSourceAIConfig_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	configKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	configName := "Terraform Test AI Config"
	configDescription := "Test AI config for data source"

	resourceName := "data.launchdarkly_ai_config.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAIConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(`
resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "%s"
	description = "%s"
	tags        = ["test"]
}

data "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = launchdarkly_ai_config.test.key
}
`, configKey, configName, configDescription)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, PROJECT_KEY),
					resource.TestCheckResourceAttrSet(resourceName, KEY),
					resource.TestCheckResourceAttrSet(resourceName, NAME),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, KEY, configKey),
					resource.TestCheckResourceAttr(resourceName, NAME, configName),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, configDescription),
					resource.TestCheckResourceAttr(resourceName, MODE, "completion"),
					resource.TestCheckResourceAttrSet(resourceName, ID),
					resource.TestCheckResourceAttrSet(resourceName, VERSION),
					resource.TestCheckResourceAttrSet(resourceName, CREATION_DATE),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
				),
			},
		},
	})
}
