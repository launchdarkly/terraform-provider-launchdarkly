package launchdarkly

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	testAccEnvironmentCreate = `
resource "launchdarkly_environment" "staging" {
	name = "Staging1"
	key = "staging1"
	color = "ff00ff"
	secure_mode = true
	default_track_events = true
	default_ttl = 50
	project_key = launchdarkly_project.test.key
	tags = ["tagged", "terraform"]
	require_comments = true
	confirm_changes = true
}
`

	testAccEnvironmentUpdate = `
resource "launchdarkly_environment" "staging" {
	name = "The real staging1"
  key = "staging1"
	color = "000000"
	secure_mode = false
	default_track_events = false
	default_ttl = 3
	project_key = launchdarkly_project.test.key
	require_comments = false
	confirm_changes = false
}
`

	testAccEnvironmentRemoveOptionalAttributes = `
resource "launchdarkly_environment" "staging" {
	name = "The real staging1"
	key = "staging1"
	color = "000000"
	project_key = launchdarkly_project.test.key
}
`

	testAccEnvironmentInvalid = `
resource "launchdarkly_environment" "staging" {
	name = "The real staging1"
	key = "staging1"
	color = "000000"
	secure_mode = false
	default_track_events = "maybe"
	default_ttl = 3
	project_key = launchdarkly_project.test.key
	require_comments = false
	confirm_changes = true
}
`

	testAccEnvironmentWithApprovals = `
resource "launchdarkly_environment" "approvals_test" {
	name = "Approvals Test"
	key = "approvals-test"
	color = "ababab"
	project_key = launchdarkly_project.test.key
	approval_settings {
		can_review_own_request = false
		min_num_approvals = 2
		required_approval_tags = ["approvals_required"]
	}
}
`
	testAccEnvironmentWithApprovalsUpdate = `
resource "launchdarkly_environment" "approvals_test" {
	name = "Approvals Test 2.0"
	key = "approvals-test"
	color = "bababa"
	project_key = launchdarkly_project.test.key
	approval_settings {
		required = true
		can_review_own_request = true
		min_num_approvals = 1
		can_apply_declined_changes = false
	}
}
`

	testAccEnvironmentWithApprovalsRemoved = `
resource "launchdarkly_environment" "approvals_test" {
	name = "Approvals Test 2.1"
	key = "approvals-test"
	color = "bababa"
	project_key = launchdarkly_project.test.key
}
`

	testAccEnvironmentCritical = `
resource "launchdarkly_environment" "critical_env" {
  name  = "Critical Approvals Env"
  key   = "critical-approvals-env"
  color = "ff00ff"
  tags  = ["terraform", "staging"]
  critical = true
  project_key = launchdarkly_project.test.key


}
`
	testAccEnvironmentCriticalUpdate = `
resource "launchdarkly_environment" "critical_env" {
  name  = "Critical Approvals Env"
  key   = "critical-approvals-env"
  color = "ff00ff"
  tags  = ["terraform", "staging"]
  critical = false
  project_key = launchdarkly_project.test.key

  approval_settings {
		required = true
		can_review_own_request = false
		min_num_approvals = 3
		can_apply_declined_changes = true
	}
}
`
)

func TestAccEnvironment_Create(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_environment.staging"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccEnvironmentCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Staging1"),
					resource.TestCheckResourceAttr(resourceName, KEY, "staging1"),
					resource.TestCheckResourceAttr(resourceName, COLOR, "ff00ff"),
					resource.TestCheckResourceAttr(resourceName, SECURE_MODE, "true"),
					resource.TestCheckResourceAttr(resourceName, DEFAULT_TRACK_EVENTS, "true"),
					resource.TestCheckResourceAttr(resourceName, DEFAULT_TTL, "50"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, REQUIRE_COMMENTS, "true"),
					resource.TestCheckResourceAttr(resourceName, CONFIRM_CHANGES, "true"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "terraform"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "tagged"),
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

func TestAccEnvironment_Update(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_environment.staging"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccEnvironmentCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Staging1"),
					resource.TestCheckResourceAttr(resourceName, KEY, "staging1"),
					resource.TestCheckResourceAttr(resourceName, COLOR, "ff00ff"),
					resource.TestCheckResourceAttr(resourceName, SECURE_MODE, "true"),
					resource.TestCheckResourceAttr(resourceName, DEFAULT_TRACK_EVENTS, "true"),
					resource.TestCheckResourceAttr(resourceName, DEFAULT_TTL, "50"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccEnvironmentUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "The real staging1"),
					resource.TestCheckResourceAttr(resourceName, KEY, "staging1"),
					resource.TestCheckResourceAttr(resourceName, COLOR, "000000"),
					resource.TestCheckResourceAttr(resourceName, SECURE_MODE, "false"),
					resource.TestCheckResourceAttr(resourceName, DEFAULT_TRACK_EVENTS, "false"),
					resource.TestCheckResourceAttr(resourceName, DEFAULT_TTL, "3"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, REQUIRE_COMMENTS, "false"),
					resource.TestCheckResourceAttr(resourceName, CONFIRM_CHANGES, "false"),
				),
			},
		},
	})
}

func TestAccEnvironment_RemoveAttributes(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_environment.staging"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccEnvironmentCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Staging1"),
					resource.TestCheckResourceAttr(resourceName, KEY, "staging1"),
					resource.TestCheckResourceAttr(resourceName, COLOR, "ff00ff"),
					resource.TestCheckResourceAttr(resourceName, SECURE_MODE, "true"),
					resource.TestCheckResourceAttr(resourceName, DEFAULT_TRACK_EVENTS, "true"),
					resource.TestCheckResourceAttr(resourceName, DEFAULT_TTL, "50"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccEnvironmentRemoveOptionalAttributes),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "The real staging1"),
					resource.TestCheckResourceAttr(resourceName, KEY, "staging1"),
					resource.TestCheckResourceAttr(resourceName, COLOR, "000000"),
					resource.TestCheckResourceAttr(resourceName, SECURE_MODE, "false"),
					resource.TestCheckResourceAttr(resourceName, DEFAULT_TRACK_EVENTS, "false"),
					resource.TestCheckResourceAttr(resourceName, DEFAULT_TTL, "0"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, REQUIRE_COMMENTS, "false"),
					resource.TestCheckResourceAttr(resourceName, CONFIRM_CHANGES, "false"),
				),
			},
		},
	})
}

func TestAccEnvironment_Invalid(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_environment.staging"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      withRandomProject(projectKey, testAccEnvironmentInvalid),
				ExpectError: regexp.MustCompile("Error: Incorrect attribute value type"), // default_track_events should be bool
			},
			{
				Config: withRandomProject(projectKey, testAccEnvironmentUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "The real staging1"),
					resource.TestCheckResourceAttr(resourceName, KEY, "staging1"),
					resource.TestCheckResourceAttr(resourceName, COLOR, "000000"),
					resource.TestCheckResourceAttr(resourceName, SECURE_MODE, "false"),
					resource.TestCheckResourceAttr(resourceName, DEFAULT_TRACK_EVENTS, "false"),
					resource.TestCheckResourceAttr(resourceName, DEFAULT_TTL, "3"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, REQUIRE_COMMENTS, "false"),
					resource.TestCheckResourceAttr(resourceName, CONFIRM_CHANGES, "false"),
				),
			},
		},
	})
}

func TestAccEnvironmentWithApprovals(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_environment.approvals_test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccEnvironmentWithApprovals),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Approvals Test"),
					resource.TestCheckResourceAttr(resourceName, KEY, "approvals-test"),
					resource.TestCheckResourceAttr(resourceName, COLOR, "ababab"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, "approval_settings.0.can_review_own_request", "false"),
					resource.TestCheckResourceAttr(resourceName, "approval_settings.0.can_apply_declined_changes", "true"), // should default to true
					resource.TestCheckResourceAttr(resourceName, "approval_settings.0.min_num_approvals", "2"),
					resource.TestCheckResourceAttr(resourceName, "approval_settings.0.required_approval_tags.0", "approvals_required"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withRandomProject(projectKey, testAccEnvironmentWithApprovalsUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Approvals Test 2.0"),
					resource.TestCheckResourceAttr(resourceName, KEY, "approvals-test"),
					resource.TestCheckResourceAttr(resourceName, COLOR, "bababa"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, "approval_settings.0.required", "true"),
					resource.TestCheckResourceAttr(resourceName, "approval_settings.0.can_review_own_request", "true"),
					resource.TestCheckResourceAttr(resourceName, "approval_settings.0.can_apply_declined_changes", "false"),
					resource.TestCheckResourceAttr(resourceName, "approval_settings.0.min_num_approvals", "1"),
					resource.TestCheckNoResourceAttr(resourceName, "approval_settings.0.required_approval_tags.#"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withRandomProject(projectKey, testAccEnvironmentWithApprovalsRemoved),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Approvals Test 2.1"),
					resource.TestCheckResourceAttr(resourceName, KEY, "approvals-test"),
					resource.TestCheckResourceAttr(resourceName, COLOR, "bababa"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckNoResourceAttr(resourceName, fmt.Sprintf("%s.%%", APPROVAL_SETTINGS)),
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

func TestAccEnvironment_Critical(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_environment.critical_env"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccEnvironmentCritical),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Critical Approvals Env"),
					resource.TestCheckResourceAttr(resourceName, KEY, "critical-approvals-env"),
					resource.TestCheckResourceAttr(resourceName, COLOR, "ff00ff"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, CRITICAL, "true"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "terraform"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "staging"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withRandomProject(projectKey, testAccEnvironmentCriticalUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Critical Approvals Env"),
					resource.TestCheckResourceAttr(resourceName, KEY, "critical-approvals-env"),
					resource.TestCheckResourceAttr(resourceName, COLOR, "ff00ff"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, CRITICAL, "false"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "terraform"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "staging"),
					resource.TestCheckResourceAttr(resourceName, "approval_settings.0.required", "true"),
					resource.TestCheckResourceAttr(resourceName, "approval_settings.0.can_review_own_request", "false"),
					resource.TestCheckResourceAttr(resourceName, "approval_settings.0.min_num_approvals", "3"),
					resource.TestCheckResourceAttr(resourceName, "approval_settings.0.can_apply_declined_changes", "true"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withRandomProject(projectKey, testAccEnvironmentCritical),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Critical Approvals Env"),
					resource.TestCheckResourceAttr(resourceName, KEY, "critical-approvals-env"),
					resource.TestCheckResourceAttr(resourceName, COLOR, "ff00ff"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, CRITICAL, "true"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "terraform"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "staging"),
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

func testAccCheckEnvironmentExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		envKey, ok := rs.Primary.Attributes[KEY]
		if !ok {
			return fmt.Errorf("environment key not found: %s", resourceName)
		}
		projKey, ok := rs.Primary.Attributes[PROJECT_KEY]
		if !ok {
			return fmt.Errorf("project key not found: %s", resourceName)
		}
		client := testAccProvider.Meta().(*Client)
		_, _, err := client.ld.EnvironmentsApi.GetEnvironment(client.ctx, projKey, envKey).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting environment. %s", err)
		}
		return nil
	}
}
