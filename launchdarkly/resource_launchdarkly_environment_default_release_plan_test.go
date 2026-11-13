package launchdarkly

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Two plans share the environment so the test can move the default between
// them. depends_on serializes their creation to avoid the API rate limit.
const testAccEnvironmentDefaultReleasePlanPlans = `
resource "launchdarkly_environment_release_plan" "first" {
	project_key     = launchdarkly_project.test.key
	environment_key = "test"
	key             = "first-plan"
	name            = "First plan"

	progressive_release = {
		randomization_unit = "user"

		phases = [
			{
				steps = [
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
}

resource "launchdarkly_environment_release_plan" "second" {
	project_key     = launchdarkly_project.test.key
	environment_key = "test"
	key             = "second-plan"
	name            = "Second plan"

	progressive_release = {
		randomization_unit = "user"

		phases = [
			{
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

	depends_on = [launchdarkly_environment_release_plan.first]
}
`

const testAccEnvironmentDefaultReleasePlanFirst = testAccEnvironmentDefaultReleasePlanPlans + `
resource "launchdarkly_environment_default_release_plan" "test" {
	project_key      = launchdarkly_project.test.key
	environment_key  = "test"
	release_plan_key = launchdarkly_environment_release_plan.first.key

	depends_on = [launchdarkly_environment_release_plan.second]
}
`

const testAccEnvironmentDefaultReleasePlanSecond = testAccEnvironmentDefaultReleasePlanPlans + `
resource "launchdarkly_environment_default_release_plan" "test" {
	project_key      = launchdarkly_project.test.key
	environment_key  = "test"
	release_plan_key = launchdarkly_environment_release_plan.second.key
}
`

func TestAccEnvironmentDefaultReleasePlan_CreateUpdate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_environment_default_release_plan.test"
	firstPlan := "launchdarkly_environment_release_plan.first"
	secondPlan := "launchdarkly_environment_release_plan.second"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEnvironmentDefaultReleasePlanDestroy,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccEnvironmentDefaultReleasePlanFirst),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckEnvironmentDefaultReleasePlanIs(resourceName, "first-plan"),
					resource.TestCheckResourceAttr(resourceName, ID, projectKey+"/test"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENVIRONMENT_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, RELEASE_PLAN_KEY, "first-plan"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// The plan resources were applied before the default was set, so
			// their is_default only reflects it after a refresh.
			{
				RefreshState: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(firstPlan, IS_DEFAULT, "true"),
					resource.TestCheckResourceAttr(secondPlan, IS_DEFAULT, "false"),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccEnvironmentDefaultReleasePlanSecond),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEnvironmentDefaultReleasePlanIs(resourceName, "second-plan"),
					resource.TestCheckResourceAttr(resourceName, ID, projectKey+"/test"),
					resource.TestCheckResourceAttr(resourceName, RELEASE_PLAN_KEY, "second-plan"),
				),
			},
			{
				RefreshState: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(firstPlan, IS_DEFAULT, "false"),
					resource.TestCheckResourceAttr(secondPlan, IS_DEFAULT, "true"),
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

// testAccCheckEnvironmentDefaultReleasePlanIs asserts via the API that the
// environment's default release plan is the expected plan key.
func testAccCheckEnvironmentDefaultReleasePlanIs(resourceName, expectedPlanKey string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		envKey := rs.Primary.Attributes[ENVIRONMENT_KEY]
		if projectKey == "" || envKey == "" {
			return fmt.Errorf("project key or environment key not found: %s", resourceName)
		}
		client := mustTestAccClient()
		plan, _, err := getEnvironmentDefaultReleasePlan(context.Background(), client, projectKey, envKey)
		if err != nil {
			return fmt.Errorf("received an error getting environment default release plan: %s", err)
		}
		if plan == nil {
			return fmt.Errorf("environment %q of project %q has no default release plan, expected %q", envKey, projectKey, expectedPlanKey)
		}
		if plan.Key != expectedPlanKey {
			return fmt.Errorf("environment %q of project %q has default release plan %q, expected %q", envKey, projectKey, plan.Key, expectedPlanKey)
		}
		return nil
	}
}

func testAccCheckEnvironmentDefaultReleasePlanDestroy(s *terraform.State) error {
	client := mustTestAccClient()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_environment_default_release_plan" {
			continue
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		envKey := rs.Primary.Attributes[ENVIRONMENT_KEY]
		plan, res, err := getEnvironmentDefaultReleasePlan(context.Background(), client, projectKey, envKey)
		// The project is destroyed with the test, so the environment may be gone too.
		if isStatusNotFound(res) {
			continue
		}
		if err != nil {
			return fmt.Errorf("unexpected error checking default release plan destruction in environment %q of project %q: %s", envKey, projectKey, err)
		}
		if plan != nil {
			return fmt.Errorf("environment %q of project %q still has default release plan %q", envKey, projectKey, plan.Key)
		}
	}
	return nil
}
