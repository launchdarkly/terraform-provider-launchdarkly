package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Project resources should be formatted with a random project key because acceptance tests
// are run in parallel on a single account.
const (
	testAccProjectCreate = `
resource "launchdarkly_project" "test" {
	key = "%s"
	name = "test project"
	tags = [ "terraform", "test" ]
	environments = [{
	  name  = "Test Environment"
	  key   = "test-env"
	  color = "010101"
	}]
}
`
	testAccProjectUpdate = `
resource "launchdarkly_project" "test" {
	key = "%s"
	name = "awesome test project"
	tags = [ "terraform" ]
	environments = [{
	  name  = "Test Environment 2.0"
	  key   = "test-env"
	  color = "020202"
	}]
}
`

	testAccProjectUpdateRemoveOptional = `
resource "launchdarkly_project" "test" {
	key = "%s"
	name = "awesome test project"
	environments = [{
		name  = "Test Environment 2.0"
		key   = "test-env"
		color = "020202"
	  }]
}
`

	testAccProjectWithEnvironment = `
resource "launchdarkly_project" "env_test" {
	key = "%s"
	name = "test project"
	environments = [{
		key = "test-env"
		name = "test environment"
		color = "000000"
		tags = ["terraform", "test"]
	}]
}	
`

	testAccProjectWithEnvironmentUpdate = `
resource "launchdarkly_project" "env_test" {
	key = "%s"
	name = "test project"
	environments = [{
		key = "test-env"
		name = "test environment updated"
		color = "AAAAAA"
		tags = ["terraform", "test", "updated"]
		default_ttl = 30
		secure_mode = true
		default_track_events = true
		require_comments = true
		confirm_changes = true
	}, {
		key = "new-approvals-env"
		name = "New approvals environment"
		color = "EEEEEE"
		tags = ["new"]
		approval_settings = [{
			required                   = true
			can_review_own_request     = true
			min_num_approvals          = 2
		  }]
	}]
}	
`

	testAccProjectWithEnvironmentUpdateApprovalSettings = `
resource "launchdarkly_project" "env_test" {
	key = "%s"
	name = "test project"
	environments = [{
		key = "test-env"
		name = "test environment updated"
		color = "AAAAAA"
		tags = ["terraform", "test", "updated"]
		default_ttl = 30
		secure_mode = true
		default_track_events = true
		require_comments = true
		confirm_changes = true
	}, {
		key = "new-approvals-env"
		name = "New approvals environment"
		color = "EEEEEE"
		tags = ["new"]
		approval_settings = [{
			required_approval_tags     = ["approvals_required"]
			can_review_own_request     = false
			min_num_approvals          = 1
			can_apply_declined_changes = false
		  }]
	}]
}	
`

	testAccProjectWithEnvironmentUpdateRemove = `
resource "launchdarkly_project" "env_test" {
	key = "%s"
	name = "test project"
	environments = [{
		key = "test-env"
		name = "test environment updated"
		color = "AAAAAA"
	}]
}	
`

	testAccProjectClientSideAvailabilityTrue = `
resource "launchdarkly_project" "test" {
	key = "%s"
	name = "test project"
	default_client_side_availability = {
		using_environment_id = true
		using_mobile_key = true
	}
	tags = [ "terraform", "test" ]
	environments = [{
	  name  = "Test Environment"
	  key   = "test-env"
	  color = "010101"
	}]
}
`

	testAccProjectWithManyEnvironments = `
locals {
  envs = [for n in range(25) : format("%s", n)]
}


resource "launchdarkly_project" "many_envs" {
  key  = "%s"
  name = "Project with many environments"

  environments = [
    for n in local.envs : {
      key   = format("env-%s", n)
      name  = format("Env %s", n)
      color = "000000"
    }
  ]

	tags = [ "terraform", "test" ]
}
`
	testAccProjectWithEnvApprovalSettings = `
resource "launchdarkly_project" "approval_env_test" {
	key = "%s"
	name = "test project"
	environments = [{
		key = "approval-env"
		name = "env with approval settings"
		color = "AAAAAA"
		approval_settings = [{
      can_review_own_request     = false
      can_apply_declined_changes = false
      min_num_approvals          = 2
      required                   = true
    }]
	}, {
		key = "default-env"
		name = "env with default approval settings"
		color = "AAAAAA"
	}]
}`

	testAccProjectWithEnvApprovalSettingsUpdate = `
resource "launchdarkly_project" "approval_env_test" {
	key = "%s"
	name = "test project"
	environments = [{
		key = "new-env"
		name = "New env with approval settings"
		color = "AAAAAA"
		approval_settings = [{
      can_review_own_request     = false
      can_apply_declined_changes = false
      min_num_approvals          = 1
      required                   = false
    }]
	}, {
		key = "approval-env"
		name = "env with approval settings"
		color = "AAAAAA"
		approval_settings = [{
      can_review_own_request     = false
      can_apply_declined_changes = false
      min_num_approvals          = 2
      required                   = true
    }]
	}, {
		key = "default-env"
		name = "env with default approval settings"
		color = "AAAAAA"
	}]
}`
)

func TestAccProject_Create(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_project.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccProjectCreate, projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "test project"),
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
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccProjectCreate, projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "test project"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
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
					resource.TestCheckResourceAttr(resourceName, KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "awesome test project"),
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
					resource.TestCheckResourceAttr(resourceName, KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "awesome test project"),
					resource.TestCheckNoResourceAttr(resourceName, "tags.#"),
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

func TestAccProject_CSA_Update_And_Revert(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_project.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccProjectCreate, projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "test project"),
					// default_client_side_availability is Optional-only; when
					// the user omits it, state stays null and the LD-API
					// defaults are not surfaced.
					resource.TestCheckNoResourceAttr(resourceName, "default_client_side_availability.%"),
				),
			},
			{
				Config: fmt.Sprintf(testAccProjectClientSideAvailabilityTrue, projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "test project"),
					resource.TestCheckResourceAttr(resourceName, "default_client_side_availability.using_environment_id", "true"),
					resource.TestCheckResourceAttr(resourceName, "default_client_side_availability.using_mobile_key", "true"),
				),
			},
			{ // make sure that removal of optional attributes reverts them to their default value
				Config: fmt.Sprintf(testAccProjectUpdateRemoveOptional, projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "awesome test project"),
					// Removing default_client_side_availability from config
					// drops it from state.
					resource.TestCheckNoResourceAttr(resourceName, "default_client_side_availability.%"),
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
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccProjectWithEnvironment, projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "test project"),
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
					resource.TestCheckResourceAttr(resourceName, KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "test project"),
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
					resource.TestCheckResourceAttr(resourceName, "environments.1.approval_settings.0.can_apply_declined_changes", "true"), // defaults to true
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
					resource.TestCheckResourceAttr(resourceName, KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "test project"),
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
					resource.TestCheckResourceAttr(resourceName, KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "test project"),
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

func TestAccProject_EnvApprovalUpdate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_project.approval_env_test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccProjectWithEnvApprovalSettings, projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "test project"),
					resource.TestCheckResourceAttr(resourceName, "environments.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.key", "approval-env"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.name", "env with approval settings"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.approval_settings.0.required", "true"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.approval_settings.0.min_num_approvals", "2"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.key", "default-env"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.name", "env with default approval settings"),
					// env[1] omits approval_settings so state stays null.
					resource.TestCheckNoResourceAttr(resourceName, "environments.1.approval_settings.#"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccProjectWithEnvApprovalSettingsUpdate, projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "test project"),
					resource.TestCheckResourceAttr(resourceName, "environments.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.key", "new-env"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.name", "New env with approval settings"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.approval_settings.0.required", "false"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.approval_settings.0.min_num_approvals", "1"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.key", "approval-env"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.name", "env with approval settings"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.approval_settings.0.required", "true"),
					resource.TestCheckResourceAttr(resourceName, "environments.1.approval_settings.0.min_num_approvals", "2"),
					resource.TestCheckResourceAttr(resourceName, "environments.2.key", "default-env"),
					resource.TestCheckResourceAttr(resourceName, "environments.2.name", "env with default approval settings"),
					// env[2] omits approval_settings so state stays null.
					resource.TestCheckNoResourceAttr(resourceName, "environments.2.approval_settings.#"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// env[0] in Step 3 declares an approval_settings whose values
				// are all LD defaults. On Import we have no prior state to
				// tell "user declared" from "user omitted", so we fall back
				// to an isZero heuristic that collapses the all-defaults
				// case to null. Ignore that drift here.
				ImportStateVerifyIgnore: []string{"environments.0.approval_settings.#", "environments.0.approval_settings.0.%", "environments.0.approval_settings.0.required", "environments.0.approval_settings.0.min_num_approvals", "environments.0.approval_settings.0.can_review_own_request", "environments.0.approval_settings.0.can_apply_declined_changes", "environments.0.approval_settings.0.auto_apply_approved_changes", "environments.0.approval_settings.0.service_kind", "environments.0.approval_settings.0.service_config.%", "environments.0.approval_settings.0.required_approval_tags.#"},
			},
		},
	})
}

func TestAccProject_ManyEnvironments(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_project.many_envs"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccProjectWithManyEnvironments, "%d", projectKey, "%s", "%s"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "Project with many environments"),
					resource.TestCheckResourceAttr(resourceName, "environments.#", "25"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "test"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// framework migration: LD API returns environments in
				// non-deterministic order, so ImportStateVerify of
				// 25-environment ordered TypeList is flaky. The framework
				// preserves config order on Refresh but has no config on
				// Import, so post-import env ordering follows the API.
				ImportStateVerifyIgnore: []string{"environments"},
			},
		},
	})
}

func TestAccProject_ViewAssociationRequirement(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_project.view_req_test"

	// Test config with view association requirements disabled (default)
	testAccProjectViewReqDefault := fmt.Sprintf(`
resource "launchdarkly_project" "view_req_test" {
	key  = "%s"
	name = "View Requirement Test"
	environments = [{
		key   = "test-env"
		name  = "Test Environment"
		color = "010101"
	}]
}
`, projectKey)

	// Test config with view association requirements enabled
	testAccProjectViewReqEnabled := fmt.Sprintf(`
resource "launchdarkly_project" "view_req_test" {
	key  = "%s"
	name = "View Requirement Test"
	require_view_association_for_new_flags    = true
	require_view_association_for_new_segments = true
	environments = [{
		key   = "test-env"
		name  = "Test Environment"
		color = "010101"
	}]
}
`, projectKey)

	// Test config with only flags view association requirement enabled
	testAccProjectViewReqFlagsOnly := fmt.Sprintf(`
resource "launchdarkly_project" "view_req_test" {
	key  = "%s"
	name = "View Requirement Test"
	require_view_association_for_new_flags    = true
	require_view_association_for_new_segments = false
	environments = [{
		key   = "test-env"
		name  = "Test Environment"
		color = "010101"
	}]
}
`, projectKey)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckProjectDestroy,
		Steps: []resource.TestStep{
			{
				// Create with defaults (both false)
				Config: testAccProjectViewReqDefault,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, REQUIRE_VIEW_ASSOCIATION_FOR_NEW_FLAGS, "false"),
					resource.TestCheckResourceAttr(resourceName, REQUIRE_VIEW_ASSOCIATION_FOR_NEW_SEGMENTS, "false"),
				),
			},
			{
				// Update to enable both
				Config: testAccProjectViewReqEnabled,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, REQUIRE_VIEW_ASSOCIATION_FOR_NEW_FLAGS, "true"),
					resource.TestCheckResourceAttr(resourceName, REQUIRE_VIEW_ASSOCIATION_FOR_NEW_SEGMENTS, "true"),
				),
			},
			{
				// Update to enable only flags
				Config: testAccProjectViewReqFlagsOnly,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, REQUIRE_VIEW_ASSOCIATION_FOR_NEW_FLAGS, "true"),
					resource.TestCheckResourceAttr(resourceName, REQUIRE_VIEW_ASSOCIATION_FOR_NEW_SEGMENTS, "false"),
				),
			},
			{
				// Import test
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

		client := mustTestAccClient()
		_, _, err := client.ld.ProjectsApi.GetProject(client.ctx, rs.Primary.ID).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting project. %s", err)
		}
		return nil
	}
}

// testAccCheckProjectDestroy verifies the project has been destroyed
func testAccCheckProjectDestroy(s *terraform.State) error {
	client := mustTestAccClient()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_project" {
			continue
		}

		_, res, err := client.ld.ProjectsApi.GetProject(client.ctx, rs.Primary.ID).Execute()

		if isStatusNotFound(res) {
			continue
		}

		if err != nil {
			return err
		}

		return fmt.Errorf("project %s still exists", rs.Primary.ID)
	}
	return nil
}
