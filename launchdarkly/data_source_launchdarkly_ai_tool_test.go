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
	testAccDataSourceAIToolBasic = `
data "launchdarkly_ai_tool" "test" {
	project_key = "%s"
	key         = "%s"
}
`
)

func TestAccDataSourceAITool_noMatchReturnsError(t *testing.T) {
	projectKey := "nonexistent-project-key"
	toolKey := "nonexistent-tool-key"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccDataSourceAIToolBasic, projectKey, toolKey),
				ExpectError: regexp.MustCompile(`failed to get AI tool with key "nonexistent-tool-key" in project "nonexistent-project-key"`),
			},
		},
	})
}

func TestAccDataSourceAITool_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	toolKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	toolDescription := "Terraform Test AI Tool"

	resourceName := "data.launchdarkly_ai_tool.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "launchdarkly_project" "test" {
	name = "AI Tool DS Test"
	key  = "%s"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_ai_tool" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	description = "%s"
	schema_json = jsonencode({
		type = "object"
		properties = {
			input = {
				type        = "string"
				description = "The input value"
			}
		}
		required = ["input"]
	})
}

data "launchdarkly_ai_tool" "test" {
	project_key = launchdarkly_project.test.key
	key         = launchdarkly_ai_tool.test.key
}
`, projectKey, toolKey, toolDescription),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, PROJECT_KEY),
					resource.TestCheckResourceAttrSet(resourceName, KEY),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, KEY, toolKey),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, toolDescription),
					resource.TestCheckResourceAttrSet(resourceName, SCHEMA_JSON),
					resource.TestCheckResourceAttrSet(resourceName, ID),
					resource.TestCheckResourceAttrSet(resourceName, VERSION),
					resource.TestCheckResourceAttrSet(resourceName, CREATION_DATE),
				),
			},
		},
	})

}
