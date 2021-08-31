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
					resource.TestCheckResourceAttr(resourceName, "name", "Staging1"),
					resource.TestCheckResourceAttr(resourceName, "key", "staging1"),
					resource.TestCheckResourceAttr(resourceName, "color", "ff00ff"),
					resource.TestCheckResourceAttr(resourceName, "secure_mode", "true"),
					resource.TestCheckResourceAttr(resourceName, "default_track_events", "true"),
					resource.TestCheckResourceAttr(resourceName, "default_ttl", "50"),
					resource.TestCheckResourceAttr(resourceName, "project_key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "require_comments", "true"),
					resource.TestCheckResourceAttr(resourceName, "confirm_changes", "true"),
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
					resource.TestCheckResourceAttr(resourceName, "name", "Staging1"),
					resource.TestCheckResourceAttr(resourceName, "key", "staging1"),
					resource.TestCheckResourceAttr(resourceName, "color", "ff00ff"),
					resource.TestCheckResourceAttr(resourceName, "secure_mode", "true"),
					resource.TestCheckResourceAttr(resourceName, "default_track_events", "true"),
					resource.TestCheckResourceAttr(resourceName, "default_ttl", "50"),
					resource.TestCheckResourceAttr(resourceName, "project_key", projectKey),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccEnvironmentUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "The real staging1"),
					resource.TestCheckResourceAttr(resourceName, "key", "staging1"),
					resource.TestCheckResourceAttr(resourceName, "color", "000000"),
					resource.TestCheckResourceAttr(resourceName, "secure_mode", "false"),
					resource.TestCheckResourceAttr(resourceName, "default_track_events", "false"),
					resource.TestCheckResourceAttr(resourceName, "default_ttl", "3"),
					resource.TestCheckResourceAttr(resourceName, "project_key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "require_comments", "false"),
					resource.TestCheckResourceAttr(resourceName, "confirm_changes", "false"),
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
					resource.TestCheckResourceAttr(resourceName, "name", "Staging1"),
					resource.TestCheckResourceAttr(resourceName, "key", "staging1"),
					resource.TestCheckResourceAttr(resourceName, "color", "ff00ff"),
					resource.TestCheckResourceAttr(resourceName, "secure_mode", "true"),
					resource.TestCheckResourceAttr(resourceName, "default_track_events", "true"),
					resource.TestCheckResourceAttr(resourceName, "default_ttl", "50"),
					resource.TestCheckResourceAttr(resourceName, "project_key", projectKey),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccEnvironmentRemoveOptionalAttributes),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "The real staging1"),
					resource.TestCheckResourceAttr(resourceName, "key", "staging1"),
					resource.TestCheckResourceAttr(resourceName, "color", "000000"),
					resource.TestCheckResourceAttr(resourceName, "secure_mode", "false"),
					resource.TestCheckResourceAttr(resourceName, "default_track_events", "false"),
					resource.TestCheckResourceAttr(resourceName, "default_ttl", "0"),
					resource.TestCheckResourceAttr(resourceName, "project_key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "require_comments", "false"),
					resource.TestCheckResourceAttr(resourceName, "confirm_changes", "false"),
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
					resource.TestCheckResourceAttr(resourceName, "name", "The real staging1"),
					resource.TestCheckResourceAttr(resourceName, "key", "staging1"),
					resource.TestCheckResourceAttr(resourceName, "color", "000000"),
					resource.TestCheckResourceAttr(resourceName, "secure_mode", "false"),
					resource.TestCheckResourceAttr(resourceName, "default_track_events", "false"),
					resource.TestCheckResourceAttr(resourceName, "default_ttl", "3"),
					resource.TestCheckResourceAttr(resourceName, "project_key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "require_comments", "false"),
					resource.TestCheckResourceAttr(resourceName, "confirm_changes", "false"),
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
		envKey, ok := rs.Primary.Attributes[KEY]
		if !ok {
			return fmt.Errorf("environment key not found: %s", resourceName)
		}
		projKey, ok := rs.Primary.Attributes[PROJECT_KEY]
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
