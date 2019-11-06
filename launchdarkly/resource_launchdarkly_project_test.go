package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const (
	testAccProjectCreate = `
resource "launchdarkly_project" "test" {
	key = "test-project"
	name = "test project"
	tags = [ "terraform", "test" ]
}
`
	testAccProjectUpdate = `
resource "launchdarkly_project" "test" {
	key = "test-project"
	name = "awesome test project"
	tags = []
}
`

	testAccProjectWithEnvironment = `
resource "launchdarkly_project" "env_test" {
	key = "test-project"
	name = "test project"
	environments {
		key = "test-env"
		name = "test environment"
		color = "000000"
		tags = ["terraform", "test"]
	}
}	
`

	testAccProjectWithEnvironmentUpdate = `
resource "launchdarkly_project" "env_test" {
	key = "test-project"
	name = "test project"
	environments {
		key = "test-env"
		name = "test environment updated"
		color = "FFFFFF"
		tags = ["updated"]
	}
}	
`
)

func TestAccProject_Create(t *testing.T) {
	resourceName := "launchdarkly_project.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", "test-project"),
					resource.TestCheckResourceAttr(resourceName, "name", "test project"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("terraform"), "terraform"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("test"), "test"),
				),
			},
		},
	})
}

func TestAccProject_Update(t *testing.T) {
	resourceName := "launchdarkly_project.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", "test-project"),
					resource.TestCheckResourceAttr(resourceName, "name", "test project"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("terraform"), "terraform"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("test"), "test"),
				),
			},
			{
				Config: testAccProjectUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", "test-project"),
					resource.TestCheckResourceAttr(resourceName, "name", "awesome test project"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "0"),
				),
			},
		},
	})
}

func TestAccProject_WithEnvironment(t *testing.T) {
	resourceName := "launchdarkly_project.env_test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectWithEnvironment,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", "test-project"),
					resource.TestCheckResourceAttr(resourceName, "name", "test project"),
					resource.TestCheckResourceAttr(resourceName, "environments.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.name", "test environment"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.tags.#", "2"),
				),
			},
			{
				Config: testAccProjectWithEnvironmentUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", "test-project"),
					resource.TestCheckResourceAttr(resourceName, "name", "test project"),
					resource.TestCheckResourceAttr(resourceName, "environments.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.name", "test environment updated"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.tags.#", "1"),
				),
			},
		},
	})
}

func testAccCheckProjectExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("project ID is not set")
		}

		client := testAccProvider.Meta().(*Client)
		_, _, err := client.ld.ProjectsApi.GetProject(client.ctx, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("received an error getting project. %s", err)
		}
		return nil
	}
}
