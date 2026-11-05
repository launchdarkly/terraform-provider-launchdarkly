package launchdarkly

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// LAUNCHDARKLY_OBSERVABILITY_ACC gates the observability filter acceptance
// tests. The beta REST endpoints they exercise are not deployed everywhere
// yet, and they need an account with observability ingest filtering enabled.
// Remove the gate once the CI account has both.
const launchdarklyObservabilityAccEnv = "LAUNCHDARKLY_OBSERVABILITY_ACC"

func testAccObservabilityPreCheck(t *testing.T) {
	if os.Getenv(launchdarklyObservabilityAccEnv) == "" {
		t.Skipf("set %s=1 to run observability filter acceptance tests (needs the beta filter endpoints and an observability-enabled account)", launchdarklyObservabilityAccEnv)
	}
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance tests skipped unless env 'TF_ACC' set")
	}
	testAccPreCheck(t)
}

func testAccObservabilityFilterConfig(projectKey, body string) string {
	return withRandomProject(projectKey, body)
}

const testAccObservabilityRulesV1 = `
resource "launchdarkly_observability_filter_rule" "drop_healthz" {
  project_key  = launchdarkly_project.test.key
  key          = "drop-healthz"
  product_type = "logs"
  name         = "Drop health checks"
  query        = "route=/healthz"
  operation    = "exclude"
}

resource "launchdarkly_observability_filter_rule" "sample_debug" {
  project_key  = launchdarkly_project.test.key
  key          = "sample-debug"
  product_type = "logs"
  name         = "Sample debug logs"
  query        = "level=debug"
  operation    = "exclude"
  rate         = 0.9
  depends_on   = [launchdarkly_observability_filter_rule.drop_healthz]
}
`

func TestAccObservabilityFilter_RuleCreateUpdateImport(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	addr := "launchdarkly_observability_filter_rule.drop_healthz"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccObservabilityPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckObservabilityFilterRulesDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccObservabilityFilterConfig(projectKey, testAccObservabilityRulesV1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "id", projectKey+"/drop-healthz"),
					resource.TestCheckResourceAttr(addr, ENABLED, "true"),
					resource.TestCheckResourceAttr(addr, RATE, "1"),
					resource.TestCheckResourceAttrSet(addr, VERSION),
				),
			},
			{
				Config: testAccObservabilityFilterConfig(projectKey, `
resource "launchdarkly_observability_filter_rule" "drop_healthz" {
  project_key  = launchdarkly_project.test.key
  key          = "drop-healthz"
  product_type = "logs"
  name         = "Drop health and readiness checks"
  query        = "route=/healthz OR route=/readyz"
  operation    = "exclude"
  enabled      = false
  rate         = 0.5
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, NAME, "Drop health and readiness checks"),
					resource.TestCheckResourceAttr(addr, ENABLED, "false"),
					resource.TestCheckResourceAttr(addr, RATE, "0.5"),
				),
			},
			{
				ResourceName:      addr,
				ImportState:       true,
				ImportStateId:     projectKey + "/drop-healthz",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccObservabilityFilter_SettingsOrderAndLimits(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	addr := "launchdarkly_observability_filter_settings.test"
	settings := func(order, limits string) string {
		return testAccObservabilityFilterConfig(projectKey, testAccObservabilityRulesV1+fmt.Sprintf(`
resource "launchdarkly_observability_filter_settings" "test" {
  project_key        = launchdarkly_project.test.key
  minute_rate_limits = %s
  rule_order = {
    logs = %s
  }
}
`, limits, order))
	}
	healthz := "launchdarkly_observability_filter_rule.drop_healthz.key"
	debug := "launchdarkly_observability_filter_rule.sample_debug.key"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccObservabilityPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckObservabilityFilterRulesDestroy,
		Steps: []resource.TestStep{
			{
				Config: settings(fmt.Sprintf("[%s, %s]", healthz, debug), "{ logs = 100000 }"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "minute_rate_limits.logs", "100000"),
					resource.TestCheckResourceAttr(addr, "rule_order.logs.#", "2"),
					resource.TestCheckResourceAttr(addr, "rule_order.logs.0", "drop-healthz"),
				),
			},
			{
				Config: settings(fmt.Sprintf("[%s, %s]", debug, healthz), "{ logs = 50000, traces = 1000 }"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "minute_rate_limits.traces", "1000"),
					resource.TestCheckResourceAttr(addr, "rule_order.logs.0", "sample-debug"),
					resource.TestCheckResourceAttr(addr, "rule_order.logs.1", "drop-healthz"),
				),
			},
			{
				ResourceName:      addr,
				ImportState:       true,
				ImportStateId:     projectKey,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckObservabilityFilterRulesDestroy(s *terraform.State) error {
	client := mustTestAccClient()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_observability_filter_rule" {
			continue
		}
		projectKey, key := rs.Primary.Attributes[PROJECT_KEY], rs.Primary.Attributes[KEY]
		_, err := getObservabilityFilterRule(context.Background(), client, projectKey, key)
		if err == nil {
			return fmt.Errorf("observability filter rule %s/%s still exists", projectKey, key)
		}
		// The project is destroyed in the same run, so a 404 for either the
		// rule or the project means the rule is gone. Anything else fails.
		if !isObservabilityNotFound(err) {
			return err
		}
	}
	return nil
}
