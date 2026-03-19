package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	testAccAIToolCreate = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "AI Tool Test Project"
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
			query = {
				type        = "string"
				description = "The search query"
			}
		}
		required = ["query"]
	})
}
`

	testAccAIToolUpdate = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "AI Tool Test Project"
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
			query = {
				type        = "string"
				description = "The search query"
			}
			limit = {
				type        = "integer"
				description = "Maximum results"
			}
		}
		required = ["query"]
	})
}
`
	testAccAIToolWithCustomParams = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "AI Tool Test Project"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_ai_tool" "test" {
	project_key       = launchdarkly_project.test.key
	key               = "%s"
	description       = "Tool with custom params"
	schema_json       = jsonencode({
		type = "object"
		properties = {
			query = { type = "string" }
		}
	})
	custom_parameters = jsonencode({
		endpoint = "https://api.example.com/search"
		timeout  = 30
	})
}
`
)

func TestAccAITool_CreateAndUpdate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	toolKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	toolDescription := "Test AI tool description"
	updatedToolDescription := "Updated AI tool description"
	resourceName := "launchdarkly_ai_tool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAIToolDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAIToolCreate, projectKey, toolKey, toolDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIToolExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, KEY, toolKey),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, toolDescription),
					resource.TestCheckResourceAttrSet(resourceName, SCHEMA_JSON),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccAIToolUpdate, projectKey, toolKey, updatedToolDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIToolExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, updatedToolDescription),
					resource.TestCheckResourceAttrSet(resourceName, SCHEMA_JSON),
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

func TestAccAITool_WithCustomParameters(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	toolKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_tool.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAIToolDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAIToolWithCustomParams, projectKey, toolKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIToolExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "Tool with custom params"),
					resource.TestCheckResourceAttrSet(resourceName, SCHEMA_JSON),
					resource.TestCheckResourceAttrSet(resourceName, CUSTOM_PARAMETERS),
					resource.TestCheckResourceAttrSet(resourceName, VERSION),
					resource.TestCheckResourceAttrSet(resourceName, CREATION_DATE),
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

func testAccCheckAIToolExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("AI tool ID is not set")
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		toolKey := rs.Primary.Attributes[KEY]

		client := testAccProvider.Meta().(*Client)
		_, _, err := client.ld.AIConfigsApi.GetAITool(client.ctx, projectKey, toolKey).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting AI tool: %s", err)
		}
		return nil
	}
}

var testAccCheckAIToolDestroy = func(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_ai_tool" {
			continue
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		toolKey := rs.Primary.Attributes[KEY]

		_, res, err := client.ld.AIConfigsApi.GetAITool(client.ctx, projectKey, toolKey).Execute()
		if isStatusNotFound(res) {
			continue
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("AI tool %s/%s still exists", projectKey, toolKey)
	}
	return nil
}
