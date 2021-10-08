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
	include_in_snippet = false
	tags = [ "terraform", "test" ]
	environments {
	  name  = "Test Environment"
	  key   = "test-env"
	  color = "010101"
	}
}
`
	testAccProjectUpdate = `
resource "launchdarkly_project" "test" {
	key = "%s"
	name = "awesome test project"
	include_in_snippet = true
	tags = [ "terraform" ]
	environments {
	  name  = "Test Environment 2.0"
	  key   = "test-env"
	  color = "020202"
	}
}
`

	testAccProjectUpdateRemoveOptional = `
resource "launchdarkly_project" "test" {
	key = "%s"
	name = "awesome test project"
	environments {
		name  = "Test Environment 2.0"
		key   = "test-env"
		color = "020202"
	  }
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
		key = "new-approvals-env"
		name = "New approvals environment"
		color = "EEEEEE"
		tags = ["new"]
		approval_settings {
			required                   = true
			can_review_own_request     = true
			min_num_approvals          = 2
		  }
	}
}	
`

	testAccProjectWithEnvironmentUpdateApprovalSettings = `
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
		key = "new-approvals-env"
		name = "New approvals environment"
		color = "EEEEEE"
		tags = ["new"]
		approval_settings {
			required_approval_tags     = ["approvals_required"]
			can_review_own_request     = false
			min_num_approvals          = 1
			can_apply_declined_changes = false
		  }
	}
}	
`

	testAccProjectWithEnvironmentUpdateRemove = `
resource "launchdarkly_project" "env_test" {
	key = "%s"
	name = "test project"
	environments {
		key = "test-env"
		name = "test environment updated"
		color = "AAAAAA"
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
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "test"),
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
					resource.TestCheckResourceAttr(resourceName, "include_in_snippet", "false"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "test"),
					resource.TestCheckResourceAttr(resourceName, "environments.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.name", "Test Environment"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.key", "test-env"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.color", "010101"),
				),
			},
			{
				Config: fmt.Sprintf(testAccProjectUpdate, projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "name", "awesome test project"),
					resource.TestCheckResourceAttr(resourceName, "include_in_snippet", "true"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
					resource.TestCheckResourceAttr(resourceName, "environments.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.name", "Test Environment 2.0"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.key", "test-env"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.color", "020202"),
				),
			},
			{ // make sure that removal of optional attributes reverts them to their null value
				Config: fmt.Sprintf(testAccProjectUpdateRemoveOptional, projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "name", "awesome test project"),
					resource.TestCheckNoResourceAttr(resourceName, "tags"),
					resource.TestCheckNoResourceAttr(resourceName, "tags.#"),
					resource.TestCheckResourceAttr(resourceName, "include_in_snippet", "false"),
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
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
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
					resource.TestCheckResourceAttr(resourceName, "environments.1.key", "new-approvals-env"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.name", "New approvals environment"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.tags.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.color", "EEEEEE"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.default_ttl", "0"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.secure_mode", "false"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.default_track_events", "false"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.require_comments", "false"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.confirm_changes", "false"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.approval_settings.0.required", "true"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.approval_settings.0.can_review_own_request", "true"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.approval_settings.0.min_num_approvals", "2"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.approval_settings.0.can_apply_declined_changes", "false"),
				),
			},
			{
				ResourceName: resourceName,
				ImportState:  true,
			},
			{
				Config: fmt.Sprintf(testAccProjectWithEnvironmentUpdateApprovalSettings, projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "name", "test project"),
					resource.TestCheckResourceAttr(resourceName, "environments.#", "2"),

					// Check approval_settings have updated as expected
					resource.TestCheckResourceAttr(resourceName, "environments.1.key", "new-approvals-env"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.name", "New approvals environment"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.tags.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.color", "EEEEEE"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.default_ttl", "0"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.secure_mode", "false"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.default_track_events", "false"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.require_comments", "false"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.confirm_changes", "false"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.approval_settings.0.required", "false"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.approval_settings.0.required_approval_tags.0", "approvals_required"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.approval_settings.0.can_review_own_request", "false"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.approval_settings.0.min_num_approvals", "1"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.approval_settings.0.can_apply_declined_changes", "false"),
				),
			},
			{
				ResourceName: resourceName,
				ImportState:  true,
			},
			{
				Config: fmt.Sprintf(testAccProjectWithEnvironmentUpdateRemove, projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "name", "test project"),
					resource.TestCheckResourceAttr(resourceName, "environments.#", "1"),

					// Check that optional attributes defaulted back to false
					resource.TestCheckResourceAttr(resourceName, "environments.0.name", "test environment updated"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.tags.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.color", "AAAAAA"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.default_ttl", "0"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.secure_mode", "false"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.default_track_events", "false"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.require_comments", "false"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.confirm_changes", "false"),
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
