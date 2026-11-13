package launchdarkly

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// The metric exists from step one so the update step can add it to metrics.
const testAccEnvironmentReleasePlanCreate = `
resource "launchdarkly_metric" "conversion" {
	project_key = launchdarkly_project.test.key
	key         = "erp-conversion"
	name        = "ERP Conversion"
	kind        = "custom"
	event_key   = "erp-conversion-event"
	is_numeric  = false
}

resource "launchdarkly_environment_release_plan" "test" {
	project_key     = launchdarkly_project.test.key
	environment_key = "test"
	key             = "standard-rollout"
	name            = "Standard rollout"
	description     = "Two-phase rollout for the test environment"

	progressive_release = {
		randomization_unit = "user"
		advancement_mode   = "manual"

		phases = [
			{
				description = "Canary"
				steps = [
					{
						allocation      = 10000
						duration_millis = 3600000
					},
				]
				clauses = [
					{
						attribute = "kind"
						op        = "in"
						values    = ["user"]
					},
				]
			},
			{
				description = "General availability"
				steps = [
					{
						allocation      = 50000
						duration_millis = 3600000
					},
					{
						allocation      = 100000
						duration_millis = 3600000
					},
				]
				clauses = [
					{
						attribute = "kind"
						op        = "in"
						values    = ["user"]
					},
				]
			},
		]
	}

	depends_on = [launchdarkly_metric.conversion]
}
`

const testAccEnvironmentReleasePlanUpdate = `
resource "launchdarkly_metric" "conversion" {
	project_key = launchdarkly_project.test.key
	key         = "erp-conversion"
	name        = "ERP Conversion"
	kind        = "custom"
	event_key   = "erp-conversion-event"
	is_numeric  = false
}

resource "launchdarkly_environment_release_plan" "test" {
	project_key     = launchdarkly_project.test.key
	environment_key = "test"
	key             = "standard-rollout"
	name            = "Standard rollout updated"

	progressive_release = {
		randomization_unit = "user"
		advancement_mode   = "automatic"
		cleanup_behavior   = "leave_rules_in_place"

		phases = [
			{
				steps = [
					{
						allocation      = 25000
						duration_millis = 7200000
					},
					{
						allocation      = 100000
						duration_millis = 7200000
					},
				]
				clauses = [
					{
						attribute = "kind"
						op        = "in"
						values    = ["user"]
					},
					{
						attribute = "email"
						op        = "endsWith"
						values    = ["@example.com"]
					},
				]
			},
		]

		metrics = {
			(launchdarkly_metric.conversion.key) = {
				is_group      = false
				auto_rollback = true
			}
		}
	}

	depends_on = [launchdarkly_metric.conversion]
}
`

func TestAccEnvironmentReleasePlan_CreateUpdate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_environment_release_plan.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfEnvironmentReleasePlansUnavailable(t, mustTestAccClient())
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEnvironmentReleasePlanDestroy,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccEnvironmentReleasePlanCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckEnvironmentReleasePlanExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ID, projectKey+"/test/standard-rollout"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENVIRONMENT_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, KEY, "standard-rollout"),
					resource.TestCheckResourceAttr(resourceName, NAME, "Standard rollout"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "Two-phase rollout for the test environment"),
					resource.TestCheckResourceAttr(resourceName, RELEASE_METHOD, RELEASE_PLAN_METHOD_PROGRESSIVE),
					resource.TestCheckResourceAttr(resourceName, VERSION, "1"),
					resource.TestCheckResourceAttr(resourceName, IS_DEFAULT, "false"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.randomization_unit", "user"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.advancement_mode", "manual"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.cleanup_behavior", CLEANUP_BEHAVIOR_REMOVE_RULES),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.phases.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.phases.0.description", "Canary"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.phases.0.steps.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.phases.0.steps.0.allocation", "10000"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.phases.0.clauses.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.phases.0.clauses.0.attribute", "kind"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.phases.0.clauses.0.context_kind", "user"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.phases.1.steps.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.phases.1.steps.1.allocation", "100000"),
					resource.TestCheckNoResourceAttr(resourceName, "progressive_release.metrics.%"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withRandomProject(projectKey, testAccEnvironmentReleasePlanUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEnvironmentReleasePlanExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Standard rollout updated"),
					resource.TestCheckNoResourceAttr(resourceName, DESCRIPTION),
					resource.TestCheckResourceAttr(resourceName, VERSION, "2"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.advancement_mode", "automatic"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.cleanup_behavior", "leave_rules_in_place"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.phases.#", "1"),
					resource.TestCheckNoResourceAttr(resourceName, "progressive_release.phases.0.description"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.phases.0.steps.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.phases.0.steps.0.allocation", "25000"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.phases.0.steps.0.duration_millis", "7200000"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.phases.0.clauses.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.phases.0.clauses.1.op", "endsWith"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.metrics.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.metrics.erp-conversion.is_group", "false"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.metrics.erp-conversion.auto_rollback", "true"),
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

// Environment release plans are enabled per account. The probe reads a plan
// that cannot exist: an entitled account answers 404, an unentitled one 403.
func skipIfEnvironmentReleasePlansUnavailable(t *testing.T, client *Client) {
	t.Helper()
	projects, _, err := client.ld.ProjectsApi.GetProjects(client.ctx).Limit(1).Expand("environments").Execute()
	if err != nil || len(projects.Items) == 0 || projects.Items[0].Environments == nil || len(projects.Items[0].Environments.Items) == 0 {
		return
	}
	project := projects.Items[0]
	_, res, err := getEnvironmentReleasePlan(context.Background(), client, project.Key, project.Environments.Items[0].Key, "entitlement-probe")
	if err != nil && res != nil && res.StatusCode == http.StatusForbidden {
		t.Skipf("environment release plans are not enabled for this account: %s", err)
	}
}

func testAccCheckEnvironmentReleasePlanExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		envKey := rs.Primary.Attributes[ENVIRONMENT_KEY]
		key := rs.Primary.Attributes[KEY]
		if projectKey == "" || envKey == "" || key == "" {
			return fmt.Errorf("project key, environment key or key not found: %s", resourceName)
		}
		client := mustTestAccClient()
		_, _, err := getEnvironmentReleasePlan(context.Background(), client, projectKey, envKey, key)
		if err != nil {
			return fmt.Errorf("received an error getting environment release plan: %s", err)
		}
		return nil
	}
}

func testAccCheckEnvironmentReleasePlanDestroy(s *terraform.State) error {
	client := mustTestAccClient()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_environment_release_plan" {
			continue
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		envKey := rs.Primary.Attributes[ENVIRONMENT_KEY]
		key := rs.Primary.Attributes[KEY]
		_, res, err := getEnvironmentReleasePlan(context.Background(), client, projectKey, envKey, key)
		if isStatusNotFound(res) {
			continue
		}
		if err != nil {
			return fmt.Errorf("unexpected error checking environment release plan %q destruction in environment %q of project %q: %s", key, envKey, projectKey, err)
		}
		return fmt.Errorf("environment release plan %q still exists in environment %q of project %q", key, envKey, projectKey)
	}
	return nil
}
