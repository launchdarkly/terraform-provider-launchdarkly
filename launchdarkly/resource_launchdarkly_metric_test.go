package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	ldapi "github.com/launchdarkly/api-client-go/v17"
	"github.com/stretchr/testify/require"
)

const (
	testAccMetricBasic = `
resource "launchdarkly_metric" "basic" {
	project_key = launchdarkly_project.test.key
	key = "basic-metric"
	name = "Basic Metric"
	description    = "Basic metric description."
	kind           = "pageview"
	tags           = [
	  "test"
	]
	urls {
	  kind = "substring"
	  substring = "foo"
	}
	urls {
		kind = "regex"
		pattern = "foo"
	  }
}
`
	testAccMetricUpdate = `
resource "launchdarkly_metric" "basic" {
	project_key = launchdarkly_project.test.key
	key = "basic-metric"
	name = "Basic updated Metric"
	description    = "Basic updated metric description."
	kind           = "pageview"
	tags           = [
	  "test"
	]
	urls {
	  kind = "substring"
	  substring = "bar"
	}
	urls {
		kind = "regex"
		pattern = "bar"
	  }
}
`

	testAccMetricCustomWithRandomizationUnitsFmt = `
resource "launchdarkly_metric" "custom" {
	project_key = "%s"
	key         = "custom-metric"
	name        = "Custom Metric"
	event_key   = "Custom event"
	kind        = "custom"
	is_numeric  = false

	randomization_units = [
		"request",
		"user"
	]
}
`

	testAccMetricCustomWithRandomizationUnitsUpdateFmt = `
resource "launchdarkly_metric" "custom" {
	project_key = "%s"
	key         = "custom-metric"
	name        = "Custom Metric"
	event_key   = "Custom event"
	kind        = "custom"
	is_numeric  = false

	randomization_units = [
		"organization",
	  "request",
		"user"
	]
}
`
)

// We can't update project experimentation settings in Terraform yet because they rely on beta endpoints. For now we will
// make individual API calls to scaffold the project, contexts, and experimentation settings.
func scaffoldProjectWithExperimentationSettings(client *Client, betaClient *Client, projectKey string, randomizationUnits []string) error {
	projectBody := ldapi.NewProjectPost(projectKey, projectKey)
	project, _, err := betaClient.ld.ProjectsApi.PostProject(betaClient.ctx).ProjectPost(*projectBody).Execute()
	if err != nil {
		return err
	}

	randomizationUnitsInput := make([]ldapi.RandomizationUnitInput, 0, len(randomizationUnits))
	for _, randomizationUnit := range randomizationUnits {
		if randomizationUnit == "user" {
			defaultTrue := true
			defaultRandomizationUnit := *ldapi.NewRandomizationUnitInput(randomizationUnit, randomizationUnit)
			defaultRandomizationUnit.Default = &defaultTrue
			randomizationUnitsInput = append(randomizationUnitsInput, defaultRandomizationUnit)
			continue
		}
		// Add the additional context kinds to the project
		contextKindPayload := ldapi.UpsertContextKindPayload{Name: randomizationUnit}
		_, _, err = client.ld.ContextsApi.PutContextKind(betaClient.ctx, project.Key, randomizationUnit).UpsertContextKindPayload(contextKindPayload).Execute()
		if err != nil {
			return err
		}
		randomizationUnitsInput = append(randomizationUnitsInput, *ldapi.NewRandomizationUnitInput(randomizationUnit, randomizationUnit))
	}

	// Update the project's experimentation settings to make the new context available for experiments
	expSettings := ldapi.RandomizationSettingsPut{
		RandomizationUnits: randomizationUnitsInput,
	}
	_, _, err = betaClient.ld.ExperimentsApi.PutExperimentationSettings(betaClient.ctx, projectKey).RandomizationSettingsPut(expSettings).Execute()
	return err
}

func TestAccMetric_BasicCreateAndUpdate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_metric.basic"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccMetricBasic),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckMetricExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Basic Metric"),
					resource.TestCheckResourceAttr(resourceName, KEY, "basic-metric"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, KIND, "pageview"),
					resource.TestCheckResourceAttr(resourceName, "urls.0.kind", "substring"),
					resource.TestCheckResourceAttr(resourceName, "urls.0.substring", "foo"),
					resource.TestCheckResourceAttr(resourceName, "urls.1.kind", "regex"),
					resource.TestCheckResourceAttr(resourceName, "urls.1.pattern", "foo"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withRandomProject(projectKey, testAccMetricUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckMetricExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Basic updated Metric"),
					resource.TestCheckResourceAttr(resourceName, KEY, "basic-metric"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, KIND, "pageview"),
					resource.TestCheckResourceAttr(resourceName, "urls.0.kind", "substring"),
					resource.TestCheckResourceAttr(resourceName, "urls.0.substring", "bar"),
					resource.TestCheckResourceAttr(resourceName, "urls.1.kind", "regex"),
					resource.TestCheckResourceAttr(resourceName, "urls.1.pattern", "bar"),
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

func TestAccMetric_WithRandomizationUnits(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_metric.custom"

	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S)
	require.NoError(t, err)
	// In order to add additional randomization units we need to update the project's context kind and
	// experimentation settings. Because this can only be done using beta endpoints we can't set this up via Terraform.
	betaClient, err := newBetaClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S)
	require.NoError(t, err)
	err = scaffoldProjectWithExperimentationSettings(client, betaClient, projectKey, []string{"user", "request", "organization"})
	require.NoError(t, err)

	defer func() {
		require.NoError(t, testAccProjectScaffoldDelete(betaClient, projectKey))
	}()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccMetricCustomWithRandomizationUnitsFmt, projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMetricExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Custom Metric"),
					resource.TestCheckResourceAttr(resourceName, KEY, "custom-metric"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, KIND, "custom"),
					resource.TestCheckResourceAttr(resourceName, EVENT_KEY, "Custom event"),
					resource.TestCheckResourceAttr(resourceName, IS_NUMERIC, "false"),
					resource.TestCheckResourceAttr(resourceName, RANDOMIZATION_UNITS+".0", "request"),
					resource.TestCheckResourceAttr(resourceName, RANDOMIZATION_UNITS+".1", "user"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccMetricCustomWithRandomizationUnitsUpdateFmt, projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMetricExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Custom Metric"),
					resource.TestCheckResourceAttr(resourceName, KEY, "custom-metric"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, KIND, "custom"),
					resource.TestCheckResourceAttr(resourceName, EVENT_KEY, "Custom event"),
					resource.TestCheckResourceAttr(resourceName, IS_NUMERIC, "false"),
					resource.TestCheckResourceAttr(resourceName, RANDOMIZATION_UNITS+".0", "organization"),
					resource.TestCheckResourceAttr(resourceName, RANDOMIZATION_UNITS+".1", "request"),
					resource.TestCheckResourceAttr(resourceName, RANDOMIZATION_UNITS+".2", "user"),
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

func TestAccMetric_MetricAnalysisFields(t *testing.T) {
	// Testing new analysis fields: INCLUDE_UNITS_WITHOUT_EVENTS, UNIT_AGGREGATION_TYPE, ANALYSIS_TYPE, PERCENTILE_VALUE

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_metric.analysis_fields"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			// 1. Set none of the analysis fields, verify the metric is created with default values
			{
				Config: withRandomProject(projectKey, `resource "launchdarkly_metric" "analysis_fields" {
	project_key = launchdarkly_project.test.key
	key = "test-analysis-fields"
	name = "Test Analysis Fields"
	description = "description."
	kind = "custom"
	event_key = "event key"
	is_numeric = true
	success_criteria = "HigherThanBaseline"
	unit = "things"
}`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMetricExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_UNITS_WITHOUT_EVENTS, "true"),
					resource.TestCheckResourceAttr(resourceName, UNIT_AGGREGATION_TYPE, "average"),
					resource.TestCheckResourceAttr(resourceName, ANALYSIS_TYPE, "mean"),
					resource.TestCheckResourceAttr(resourceName, PERCENTILE_VALUE, "0"),
					resource.TestCheckResourceAttr(resourceName, VERSION, "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},

			// 2. Run again with same config, verify version does not increment
			{
				Config: withRandomProject(projectKey, `resource "launchdarkly_metric" "analysis_fields" {
	project_key = launchdarkly_project.test.key
	key = "test-analysis-fields"
	name = "Test Analysis Fields"
	description = "description."
	kind = "custom"
	event_key = "event key"
	is_numeric = true
	success_criteria = "HigherThanBaseline"
	unit = "things"
}`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMetricExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_UNITS_WITHOUT_EVENTS, "true"),
					resource.TestCheckResourceAttr(resourceName, UNIT_AGGREGATION_TYPE, "average"),
					resource.TestCheckResourceAttr(resourceName, ANALYSIS_TYPE, "mean"),
					resource.TestCheckResourceAttr(resourceName, PERCENTILE_VALUE, "0"),
					resource.TestCheckResourceAttr(resourceName, VERSION, "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},

			// 3. Set all analysis fields to their default values, verify version is still 1 (no update happened)
			{
				Config: withRandomProject(projectKey, `resource "launchdarkly_metric" "analysis_fields" {
	project_key = launchdarkly_project.test.key
	key = "test-analysis-fields"
	name = "Test Analysis Fields"
	description = "description."
	kind = "custom"
	event_key = "event key"
	is_numeric = true
	success_criteria = "HigherThanBaseline"
	unit = "things"
	include_units_without_events = true
	unit_aggregation_type = "average"
	analysis_type = "mean"
	percentile_value = null
}`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMetricExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_UNITS_WITHOUT_EVENTS, "true"),
					resource.TestCheckResourceAttr(resourceName, UNIT_AGGREGATION_TYPE, "average"),
					resource.TestCheckResourceAttr(resourceName, ANALYSIS_TYPE, "mean"),
					resource.TestCheckResourceAttr(resourceName, PERCENTILE_VALUE, "0"),
					resource.TestCheckResourceAttr(resourceName, VERSION, "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// 4. Set analysis_type to percentile and leave percentile blank. verify error.
			{
				Config: withRandomProject(projectKey, `resource "launchdarkly_metric" "analysis_fields" {
	project_key = launchdarkly_project.test.key
	key = "test-analysis-fields"
	name = "Test Analysis Fields"
	description = "description."
	kind = "custom"
	event_key = "event key"
	is_numeric = true
	success_criteria = "HigherThanBaseline"
	unit = "things"
	include_units_without_events = false
	unit_aggregation_type = "sum"
	analysis_type = "percentile"
	percentile_value = null
}`),
				ExpectError: regexp.MustCompile("percentile_value is required when analysis_type is percentile"),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// 5. Set percentile_value, verify metric is updated. (version is now 2)
			{
				Config: withRandomProject(projectKey, `resource "launchdarkly_metric" "analysis_fields" {
	project_key = launchdarkly_project.test.key
	key = "test-analysis-fields"
	name = "Test Analysis Fields"
	description = "description."
	kind = "custom"
	event_key = "event key"
	is_numeric = true
	success_criteria = "HigherThanBaseline"
	unit = "things"
	include_units_without_events = false
	unit_aggregation_type = "sum"
	analysis_type = "percentile"
	percentile_value = 42
}`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMetricExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_UNITS_WITHOUT_EVENTS, "false"),
					resource.TestCheckResourceAttr(resourceName, UNIT_AGGREGATION_TYPE, "sum"),
					resource.TestCheckResourceAttr(resourceName, ANALYSIS_TYPE, "percentile"),
					resource.TestCheckResourceAttr(resourceName, PERCENTILE_VALUE, "42"),
					resource.TestCheckResourceAttr(resourceName, VERSION, "2"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// 6. Change percentile_value (version is now 3)
			{
				Config: withRandomProject(projectKey, `resource "launchdarkly_metric" "analysis_fields" {
	project_key = launchdarkly_project.test.key
	key = "test-analysis-fields"
	name = "Test Analysis Fields"
	description = "description."
	kind = "custom"
	event_key = "event key"
	is_numeric = true
	success_criteria = "HigherThanBaseline"
	unit = "things"
	include_units_without_events = false
	unit_aggregation_type = "sum"
	analysis_type = "percentile"
	percentile_value = 99
}`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMetricExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_UNITS_WITHOUT_EVENTS, "false"),
					resource.TestCheckResourceAttr(resourceName, UNIT_AGGREGATION_TYPE, "sum"),
					resource.TestCheckResourceAttr(resourceName, ANALYSIS_TYPE, "percentile"),
					resource.TestCheckResourceAttr(resourceName, PERCENTILE_VALUE, "99"),
					resource.TestCheckResourceAttr(resourceName, VERSION, "3"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// 7. Change key, verify old metric is deleted, new one is created, fields all correct. (version is now 1)
			{
				Config: withRandomProject(projectKey, `resource "launchdarkly_metric" "analysis_fields" {
	project_key = launchdarkly_project.test.key
	key = "test-analysis-fields2"
	name = "Test Analysis Fields"
	description = "description."
	kind = "custom"
	event_key = "event key"
	is_numeric = true
	success_criteria = "HigherThanBaseline"
	unit = "things"
	include_units_without_events = false
	unit_aggregation_type = "sum"
	analysis_type = "percentile"
	percentile_value = 99
}`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMetricExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_UNITS_WITHOUT_EVENTS, "false"),
					resource.TestCheckResourceAttr(resourceName, UNIT_AGGREGATION_TYPE, "sum"),
					resource.TestCheckResourceAttr(resourceName, ANALYSIS_TYPE, "percentile"),
					resource.TestCheckResourceAttr(resourceName, PERCENTILE_VALUE, "99"),
					resource.TestCheckResourceAttr(resourceName, VERSION, "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// 8. Change analysis type, verify error
			{
				Config: withRandomProject(projectKey, `resource "launchdarkly_metric" "analysis_fields" {
	project_key = launchdarkly_project.test.key
	key = "test-analysis-fields2"
	name = "Test Analysis Fields"
	description = "description."
	kind = "custom"
	event_key = "event key"
	is_numeric = true
	success_criteria = "HigherThanBaseline"
	unit = "things"
	include_units_without_events = false
	unit_aggregation_type = "sum"
	analysis_type = "mean"
	percentile_value = 99
}`),
				ExpectError: regexp.MustCompile("mean type metrics can not have percentile values"),
			},
			// 9. Remove percentile, verify metric is updated. (version is now 2)
			{
				Config: withRandomProject(projectKey, `resource "launchdarkly_metric" "analysis_fields" {
	project_key = launchdarkly_project.test.key
	key = "test-analysis-fields2"
	name = "Test Analysis Fields"
	description = "description."
	kind = "custom"
	event_key = "event key"
	is_numeric = true
	success_criteria = "HigherThanBaseline"
	unit = "things"
	include_units_without_events = false
	unit_aggregation_type = "sum"
	analysis_type = "mean"
}`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMetricExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_UNITS_WITHOUT_EVENTS, "false"),
					resource.TestCheckResourceAttr(resourceName, UNIT_AGGREGATION_TYPE, "sum"),
					resource.TestCheckResourceAttr(resourceName, ANALYSIS_TYPE, "mean"),
					resource.TestCheckResourceAttr(resourceName, PERCENTILE_VALUE, "0"),
					resource.TestCheckResourceAttr(resourceName, VERSION, "2"),
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

func TestAccMetric_IncludeUnitsWithoutEvents(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_metric.analysis_fields"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			// Default value is "true" when "analysis_type" is "mean"
			{
				Config: withRandomProject(projectKey, `resource "launchdarkly_metric" "analysis_fields" {
	project_key = launchdarkly_project.test.key
	key = "test-analysis-fields"
	name = "Test Analysis Fields"
	description = "description."
	kind = "custom"
	event_key = "event key"
	is_numeric = true
	success_criteria = "HigherThanBaseline"
	unit = "things"
	analysis_type = "mean"
}`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMetricExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_UNITS_WITHOUT_EVENTS, "true"),
					resource.TestCheckResourceAttr(resourceName, VERSION, "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Default value is "false" when "analysis_type" is "percentile"
			{
				Config: withRandomProject(projectKey, `resource "launchdarkly_metric" "analysis_fields" {
	project_key = launchdarkly_project.test.key
	key = "test-analysis-fields"
	name = "Test Analysis Fields"
	description = "description."
	kind = "custom"
	event_key = "event key"
	is_numeric = true
	success_criteria = "HigherThanBaseline"
	unit = "things"
	analysis_type = "percentile"
	percentile_value = 99
}`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMetricExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_UNITS_WITHOUT_EVENTS, "false"),
					resource.TestCheckResourceAttr(resourceName, VERSION, "2"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// "false" is also allowed when "analysis_type" is "mean"
			{
				Config: withRandomProject(projectKey, `resource "launchdarkly_metric" "analysis_fields" {
	project_key = launchdarkly_project.test.key
	key = "test-analysis-fields"
	name = "Test Analysis Fields"
	description = "description."
	kind = "custom"
	event_key = "event key"
	is_numeric = true
	success_criteria = "HigherThanBaseline"
	unit = "things"
	analysis_type = "mean"
	include_units_without_events = false
}`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMetricExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_UNITS_WITHOUT_EVENTS, "false"),
					resource.TestCheckResourceAttr(resourceName, VERSION, "3"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// "true" is not allowed when "analysis_type" is "percentile"
			{
				Config: withRandomProject(projectKey, `resource "launchdarkly_metric" "analysis_fields" {
	project_key = launchdarkly_project.test.key
	key = "test-analysis-fields"
	name = "Test Analysis Fields"
	description = "description."
	kind = "custom"
	event_key = "event key"
	is_numeric = true
	success_criteria = "HigherThanBaseline"
	unit = "things"
	analysis_type = "percentile"
	percentile_value = 99
	include_units_without_events = true
}`),
				ExpectError: regexp.MustCompile("include_units_without_events is not supported for percentile metrics"),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckMetricExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		metricKey, ok := rs.Primary.Attributes[KEY]
		if !ok {
			return fmt.Errorf("metric key not found: %s", resourceName)
		}
		projKey, ok := rs.Primary.Attributes[PROJECT_KEY]
		if !ok {
			return fmt.Errorf("project key not found: %s", resourceName)
		}
		client := testAccProvider.Meta().(*Client)
		_, _, err := client.ld.MetricsApi.GetMetric(client.ctx, projKey, metricKey).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting metric. %s", err)
		}
		return nil
	}
}
