package launchdarkly

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	ldapi "github.com/launchdarkly/api-client-go/v22"
	"github.com/stretchr/testify/require"
)

// testAccExperimentScaffold creates the project (with experimentation settings),
// a boolean flag, and a metric that the experiment references. Experiment
// treatments must reference real flag variation IDs, which are server-assigned,
// so the supporting resources are created via the API rather than Terraform.
// It returns the environment key, flag key, metric key, the two boolean
// variation IDs, and the flag's environment configuration version.
func testAccExperimentScaffold(t *testing.T, client, betaClient *Client, projectKey string) (envKey, flagKey, metricKey, controlVarID, treatmentVarID string, flagConfigVersion int32) {
	t.Helper()
	envKey = "production"

	require.NoError(t, scaffoldProjectWithExperimentationSettings(client, betaClient, projectKey, []string{"user"}))

	flagKey = "experiment-flag"
	flagBody := ldapi.NewFeatureFlagBody("Experiment flag", flagKey)
	flag, _, err := client.ld.FeatureFlagsApi.PostFeatureFlag(client.ctx, projectKey).FeatureFlagBody(*flagBody).Execute()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(flag.Variations), 2)
	controlVarID = *flag.Variations[0].Id
	treatmentVarID = *flag.Variations[1].Id
	if flag.Environments != nil {
		if cfg, ok := (*flag.Environments)[envKey]; ok {
			flagConfigVersion = cfg.Version
		}
	}
	if flagConfigVersion == 0 {
		flagConfigVersion = 1
	}

	metricKey = "experiment-metric"
	metricName := "Experiment metric"
	eventKey := "purchase"
	metricPost := ldapi.MetricPost{
		Key:      metricKey,
		Name:     &metricName,
		Kind:     "custom",
		EventKey: &eventKey,
	}
	_, _, err = client.ld.MetricsApi.PostMetric(client.ctx, projectKey).MetricPost(metricPost).Execute()
	require.NoError(t, err)

	return envKey, flagKey, metricKey, controlVarID, treatmentVarID, flagConfigVersion
}

func testAccExperimentConfig(projectKey, envKey, flagKey, metricKey, controlVarID, treatmentVarID, name string, flagConfigVersion int32) string {
	return fmt.Sprintf(`
resource "launchdarkly_experiment" "test" {
	project_key     = "%s"
	environment_key = "%s"
	key             = "checkout-experiment"
	name            = "%s"
	description     = "Acceptance test experiment."

	iteration = {
		hypothesis                = "The treatment increases conversions."
		randomization_unit        = "user"
		primary_single_metric_key = "%s"

		metrics = [{
			key = "%s"
		}]

		treatments = [
			{
				name               = "Control"
				baseline           = true
				allocation_percent = "50"
				parameters = [{
					flag_key     = "%s"
					variation_id = "%s"
				}]
			},
			{
				name               = "Treatment"
				baseline           = false
				allocation_percent = "50"
				parameters = [{
					flag_key     = "%s"
					variation_id = "%s"
				}]
			},
		]

		flags = {
			"%s" = {
				rule_id             = "fallthrough"
				flag_config_version = %d
			}
		}
	}
}
`, projectKey, envKey, name, metricKey, metricKey, flagKey, controlVarID, flagKey, treatmentVarID, flagKey, flagConfigVersion)
}

func TestAccExperiment_CreateUpdate(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.SkipNow()
	}
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_experiment.test"

	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)
	betaClient, err := newBetaClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)

	envKey, flagKey, metricKey, controlVarID, treatmentVarID, flagConfigVersion := testAccExperimentScaffold(t, client, betaClient, projectKey)

	defer func() {
		require.NoError(t, testAccProjectScaffoldDelete(betaClient, projectKey))
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckExperimentArchived(client),
		Steps: []resource.TestStep{
			{
				Config: testAccExperimentConfig(projectKey, envKey, flagKey, metricKey, controlVarID, treatmentVarID, "Checkout experiment", flagConfigVersion),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, NAME, "Checkout experiment"),
					resource.TestCheckResourceAttr(resourceName, KEY, "checkout-experiment"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENVIRONMENT_KEY, envKey),
					resource.TestCheckResourceAttr(resourceName, "iteration.hypothesis", "The treatment increases conversions."),
					resource.TestCheckResourceAttr(resourceName, "archived", "false"),
				),
			},
			{
				Config: testAccExperimentConfig(projectKey, envKey, flagKey, metricKey, controlVarID, treatmentVarID, "Checkout experiment updated", flagConfigVersion),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, NAME, "Checkout experiment updated"),
				),
			},
		},
	})
}

// testAccCheckExperimentArchived verifies that destroyed experiments are
// archived (the LaunchDarkly API has no delete endpoint, so destroy archives).
func testAccCheckExperimentArchived(client *Client) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "launchdarkly_experiment" {
				continue
			}
			projectKey, environmentKey, key, err := experimentIDToKeys(rs.Primary.ID)
			if err != nil {
				return err
			}
			experiment, _, err := client.ld.ExperimentsApi.GetExperiment(client.ctx, projectKey, environmentKey, key).Execute()
			if err != nil {
				return fmt.Errorf("failed to get experiment %q during destroy check: %s", key, handleLdapiErr(err).Error())
			}
			if experiment.ArchivedDate == nil {
				return fmt.Errorf("experiment %q was not archived on destroy", key)
			}
		}
		return nil
	}
}
