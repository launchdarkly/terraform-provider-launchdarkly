package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Project resources should be formatted with a random project key because acceptance tests
// are run in parallel on a single account.
const (
	testAccProjectCreate = `
resource "launchdarkly_project" "test" {
	key = "%s"
	name = "test project"
	tags = [ "terraform", "test" ]
}
`
	testAccProjectUpdate = `
resource "launchdarkly_project" "test" {
	key = "%s"
	name = "awesome test project"
	include_in_snippet = true
	tags = []
}
`

	testAccProjectWithEnvironment = `
resource "launchdarkly_project" "env_test" {
	key = "%s"
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
	key = "%s"
	name = "test project"
	environments {
		key = "test-env"
		name = "test environment updated"
		color = "AAAAAA"
		tags = ["terraform", "test", "updated"]
		default_ttl = 30
		secure_mode = true
		default_track_events = true
		require_comments = true
		confirm_changes = true
	}

	environments {
		key = "new-env"
		name = "New test environment"
		color = "EEEEEE"
		tags = ["new"]
	}
}
`
)

func TestAccProject_Create(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_project.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccProjectCreate, projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "name", "test project"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("terraform"), "terraform"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("test"), "test"),
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

func TestAccProject_Update(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_project.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccProjectCreate, projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "name", "test project"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("terraform"), "terraform"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("test"), "test"),
				),
			},
			{
				Config: fmt.Sprintf(testAccProjectUpdate, projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "name", "awesome test project"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "include_in_snippet", "true"),
				),
			},
		},
	})
}

func TestAccProject_WithEnvironments(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_project.env_test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccProjectWithEnvironment, projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "name", "test project"),
					resource.TestCheckResourceAttr(resourceName, "environments.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.name", "test environment"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.color", "000000"),

					// default environment values
					resource.TestCheckResourceAttr(resourceName, "environments.0.default_ttl", "0"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.secure_mode", "false"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.default_track_events", "false"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.require_comments", "false"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.confirm_changes", "false"),
				),
			},
			{
				ResourceName: resourceName,
				ImportState:  true,
			},
			{
				Config: fmt.Sprintf(testAccProjectWithEnvironmentUpdate, projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "name", "test project"),
					resource.TestCheckResourceAttr(resourceName, "environments.#", "2"),

					// Check environment 0 was updated
					resource.TestCheckResourceAttr(resourceName, "environments.0.name", "test environment updated"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.tags.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.color", "AAAAAA"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.default_ttl", "30"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.secure_mode", "true"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.default_track_events", "true"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.require_comments", "true"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.confirm_changes", "true"),

					// Check environment 1 is created
					resource.TestCheckResourceAttr(resourceName, "environments.1.name", "New test environment"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.tags.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.color", "EEEEEE"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.default_ttl", "0"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.secure_mode", "false"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.default_track_events", "false"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.require_comments", "false"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.confirm_changes", "false"),
				),
			},
			{
				ResourceName: resourceName,
				ImportState:  true,
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
