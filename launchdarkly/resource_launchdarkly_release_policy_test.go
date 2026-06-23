package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// A guarded release policy monitors a metric during rollout. The metric is
// created first and referenced by key; depends_on serializes creation to avoid
// tripping the account-wide API rate limit.
const testAccReleasePolicyGuardedFmt = `
resource "launchdarkly_metric" "conversion" {
	project_key = launchdarkly_project.test.key
	key         = "rp-conversion"
	name        = "RP Conversion"
	kind        = "custom"
	event_key   = "rp-conversion-event"
	is_numeric  = false
}

resource "launchdarkly_release_policy" "test" {
	project_key    = launchdarkly_project.test.key
	key            = "guarded-rollout"
	name           = "Guarded rollout"
	release_method = "guarded-release"

	scope = {
		environment_keys = ["test"]
	}

	guarded_release_config = {
		rollout_context_kind   = "user"
		min_sample_size        = 100
		rollback_on_regression = true
		metric_keys            = [launchdarkly_metric.conversion.key]

		stages = [
			{
				allocation      = 10
				duration_millis = 3600000
			},
			{
				allocation      = 50
				duration_millis = 3600000
			},
		]
	}

	depends_on = [launchdarkly_metric.conversion]
}
`

const testAccReleasePolicyGuardedUpdateFmt = `
resource "launchdarkly_metric" "conversion" {
	project_key = launchdarkly_project.test.key
	key         = "rp-conversion"
	name        = "RP Conversion"
	kind        = "custom"
	event_key   = "rp-conversion-event"
	is_numeric  = false
}

resource "launchdarkly_release_policy" "test" {
	project_key    = launchdarkly_project.test.key
	key            = "guarded-rollout"
	name           = "Guarded rollout updated"
	release_method = "guarded-release"

	scope = {
		environment_keys = ["test"]
	}

	guarded_release_config = {
		rollout_context_kind   = "user"
		min_sample_size        = 250
		rollback_on_regression = false
		metric_keys            = [launchdarkly_metric.conversion.key]

		stages = [
			{
				allocation      = 25
				duration_millis = 7200000
			},
		]
	}

	depends_on = [launchdarkly_metric.conversion]
}
`

const testAccReleasePolicyProgressiveFmt = `
resource "launchdarkly_release_policy" "test" {
	project_key    = launchdarkly_project.test.key
	key            = "progressive-rollout"
	name           = "Progressive rollout"
	release_method = "progressive-release"

	scope = {
		environment_keys = ["test"]
	}

	progressive_release_config = {
		rollout_context_kind = "user"

		stages = [
			{
				allocation      = 20
				duration_millis = 3600000
			},
			{
				allocation      = 60
				duration_millis = 3600000
			},
		]
	}
}
`

func TestAccReleasePolicy_Guarded(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_release_policy.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckReleasePolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccReleasePolicyGuardedFmt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckReleasePolicyExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Guarded rollout"),
					resource.TestCheckResourceAttr(resourceName, KEY, "guarded-rollout"),
					resource.TestCheckResourceAttr(resourceName, RELEASE_METHOD, "guarded-release"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, "scope.environment_keys.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "guarded_release_config.min_sample_size", "100"),
					resource.TestCheckResourceAttr(resourceName, "guarded_release_config.rollback_on_regression", "true"),
					resource.TestCheckResourceAttr(resourceName, "guarded_release_config.metric_keys.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "guarded_release_config.stages.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "guarded_release_config.stages.0.allocation", "10"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withRandomProject(projectKey, testAccReleasePolicyGuardedUpdateFmt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckReleasePolicyExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Guarded rollout updated"),
					resource.TestCheckResourceAttr(resourceName, "guarded_release_config.min_sample_size", "250"),
					resource.TestCheckResourceAttr(resourceName, "guarded_release_config.rollback_on_regression", "false"),
					resource.TestCheckResourceAttr(resourceName, "guarded_release_config.stages.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "guarded_release_config.stages.0.allocation", "25"),
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

func TestAccReleasePolicy_Progressive(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_release_policy.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckReleasePolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccReleasePolicyProgressiveFmt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckReleasePolicyExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, RELEASE_METHOD, "progressive-release"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release_config.rollout_context_kind", "user"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release_config.stages.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release_config.stages.1.allocation", "60"),
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

func testAccCheckReleasePolicyExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		key, ok := rs.Primary.Attributes[KEY]
		if !ok {
			return fmt.Errorf("release policy key not found: %s", resourceName)
		}
		projKey, ok := rs.Primary.Attributes[PROJECT_KEY]
		if !ok {
			return fmt.Errorf("project key not found: %s", resourceName)
		}
		beta, err := newReleasePolicyBetaClient(mustTestAccClient())
		if err != nil {
			return err
		}
		_, _, err = beta.ld.ReleasePoliciesBetaApi.GetReleasePolicy(beta.ctx, projKey, key).
			LDAPIVersion(RELEASE_POLICY_BETA_VERSION).
			Execute()
		if err != nil {
			return fmt.Errorf("received an error getting release policy: %s", err)
		}
		return nil
	}
}

func testAccCheckReleasePolicyDestroy(s *terraform.State) error {
	beta, err := newReleasePolicyBetaClient(mustTestAccClient())
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_release_policy" {
			continue
		}
		projKey := rs.Primary.Attributes[PROJECT_KEY]
		key := rs.Primary.Attributes[KEY]
		_, res, err := beta.ld.ReleasePoliciesBetaApi.GetReleasePolicy(beta.ctx, projKey, key).
			LDAPIVersion(RELEASE_POLICY_BETA_VERSION).
			Execute()
		if isStatusNotFound(res) {
			continue
		}
		if err != nil {
			return fmt.Errorf("unexpected error checking release policy %q destruction in project %q: %s", key, projKey, handleLdapiErr(err))
		}
		return fmt.Errorf("release policy %q still exists in project %q", key, projKey)
	}
	return nil
}
