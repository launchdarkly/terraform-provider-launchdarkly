package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

const (
	testAccEnvironmentCreate = `
resource "launchdarkly_project" "test" {
	name = "testProject"
	key = "test-project"
}

resource "launchdarkly_environment" "staging" {
	name = "Staging1"
  	key = "staging1"
  	color = "ff00ff"
  	secure_mode = true
  	default_track_events = false
  	default_ttl = 50
  	project_key = launchdarkly_project.test.key
}
`
	testAccEnvironmentUpdate = `
resource "launchdarkly_project" "test" {
	name = "testProject"
	key = "test-project"
}

resource "launchdarkly_environment" "staging" {
	name = "The real staging1"
  	key = "staging1"
  	color = "000000"
  	secure_mode = false
  	default_track_events = true
  	default_ttl = 3
  	project_key = launchdarkly_project.test.key
}
`
)

func TestAccEnvironment_Create(t *testing.T) {
	resourceName := "launchdarkly_environment.staging"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "Staging1"),
					resource.TestCheckResourceAttr(resourceName, "key", "staging1"),
					resource.TestCheckResourceAttr(resourceName, "color", "ff00ff"),
					resource.TestCheckResourceAttr(resourceName, "secure_mode", "true"),
					resource.TestCheckResourceAttr(resourceName, "default_track_events", "false"),
					resource.TestCheckResourceAttr(resourceName, "default_ttl", "50"),
					resource.TestCheckResourceAttr(resourceName, "project_key", "test-project"),
				),
			},
		},
	})
}

func TestAccEnvironment_Update(t *testing.T) {
	resourceName := "launchdarkly_environment.staging"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "Staging1"),
					resource.TestCheckResourceAttr(resourceName, "key", "staging1"),
					resource.TestCheckResourceAttr(resourceName, "color", "ff00ff"),
					resource.TestCheckResourceAttr(resourceName, "secure_mode", "true"),
					resource.TestCheckResourceAttr(resourceName, "default_track_events", "false"),
					resource.TestCheckResourceAttr(resourceName, "default_ttl", "50"),
					resource.TestCheckResourceAttr(resourceName, "project_key", "test-project"),
				),
			},
			{
				Config: testAccEnvironmentUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "The real staging1"),
					resource.TestCheckResourceAttr(resourceName, "key", "staging1"),
					resource.TestCheckResourceAttr(resourceName, "color", "000000"),
					resource.TestCheckResourceAttr(resourceName, "secure_mode", "false"),
					resource.TestCheckResourceAttr(resourceName, "default_track_events", "true"),
					resource.TestCheckResourceAttr(resourceName, "default_ttl", "3"),
					resource.TestCheckResourceAttr(resourceName, "project_key", "test-project"),
				),
			},
		},
	})
}

func testAccCheckEnvironmentExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		envKey, ok := rs.Primary.Attributes[key]
		if !ok {
			return fmt.Errorf("environment key not found: %s", resourceName)
		}
		projKey, ok := rs.Primary.Attributes[project_key]
		if !ok {
			return fmt.Errorf("project key not found: %s", resourceName)
		}
		client := testAccProvider.Meta().(*Client)
		_, _, err := client.ld.EnvironmentsApi.GetEnvironment(client.ctx, projKey, envKey)
		if err != nil {
			return fmt.Errorf("received an error getting environment. %s", err)
		}
		return nil
	}
}
