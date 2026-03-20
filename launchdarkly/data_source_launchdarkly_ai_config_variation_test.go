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
	testAccDataSourceAIConfigVariationBasic = `
data "launchdarkly_ai_config_variation" "test" {
	project_key = "%s"
	config_key  = "%s"
	key         = "%s"
}
`
)

func TestAccDataSourceAIConfigVariation_noMatchReturnsError(t *testing.T) {
	projectKey := "nonexistent-project-key"
	configKey := "nonexistent-config-key"
	variationKey := "nonexistent-variation-key"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAIConfigVariationDestroy,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccDataSourceAIConfigVariationBasic, projectKey, configKey, variationKey),
				ExpectError: regexp.MustCompile(`failed to get AI config variation with key "nonexistent-variation-key"`),
			},
		},
	})
}

func TestAccDataSourceAIConfigVariation_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	configKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	variationKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resourceName := "data.launchdarkly_ai_config_variation.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAIConfigVariationDestroy,
		Steps: []resource.TestStep{
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(`
resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "Parent AI Config"
	description = "Parent config for data source test"
	tags        = ["test"]
}

resource "launchdarkly_ai_config_variation" "test" {
	project_key = launchdarkly_project.test.key
	config_key  = launchdarkly_ai_config.test.key
	key         = "%s"
	name        = "Test Variation"
	messages {
		role    = "system"
		content = "You are a helpful assistant."
	}
}

data "launchdarkly_ai_config_variation" "test" {
	project_key = launchdarkly_project.test.key
	config_key  = launchdarkly_ai_config.test.key
	key         = launchdarkly_ai_config_variation.test.key
}
`, configKey, variationKey)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, PROJECT_KEY),
					resource.TestCheckResourceAttrSet(resourceName, AI_CONFIG_KEY),
					resource.TestCheckResourceAttrSet(resourceName, KEY),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, AI_CONFIG_KEY, configKey),
					resource.TestCheckResourceAttr(resourceName, KEY, variationKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "Test Variation"),
					resource.TestCheckResourceAttrSet(resourceName, VARIATION_ID),
					resource.TestCheckResourceAttrSet(resourceName, VERSION),
					resource.TestCheckResourceAttrSet(resourceName, CREATION_DATE),
					resource.TestCheckResourceAttr(resourceName, "messages.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "messages.0.role", "system"),
					resource.TestCheckResourceAttr(resourceName, "messages.0.content", "You are a helpful assistant."),
				),
			},
		},
	})
}
