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
	testAccDataSourceModelConfigBasic = `
data "launchdarkly_model_config" "test" {
	project_key = "%s"
	key         = "%s"
}
`
)

func TestAccDataSourceModelConfig_noMatchReturnsError(t *testing.T) {
	projectKey := "nonexistent-project-key"
	modelConfigKey := "nonexistent-model-config-key"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccDataSourceModelConfigBasic, projectKey, modelConfigKey),
				ExpectError: regexp.MustCompile(`failed to get model config with key "nonexistent-model-config-key" in project "nonexistent-project-key"`),
			},
		},
	})
}

func TestAccDataSourceModelConfig_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	modelConfigKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	modelConfigName := "Terraform Test Model Config"
	modelID := "gpt-4"
	providerName := "openai"

	resourceName := "data.launchdarkly_model_config.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "launchdarkly_project" "test" {
	name = "Model Config DS Test"
	key  = "%s"
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
	provider    = "%s"
	tags        = ["test"]
}

data "launchdarkly_model_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = launchdarkly_model_config.test.key
}
`, projectKey, modelConfigKey, modelConfigName, modelID, providerName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, PROJECT_KEY),
					resource.TestCheckResourceAttrSet(resourceName, KEY),
					resource.TestCheckResourceAttrSet(resourceName, NAME),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, KEY, modelConfigKey),
					resource.TestCheckResourceAttr(resourceName, NAME, modelConfigName),
					resource.TestCheckResourceAttr(resourceName, MODEL_ID, modelID),
					resource.TestCheckResourceAttr(resourceName, PROVIDER_NAME, providerName),
					resource.TestCheckResourceAttrSet(resourceName, ID),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
				),
			},
		},
	})

}
