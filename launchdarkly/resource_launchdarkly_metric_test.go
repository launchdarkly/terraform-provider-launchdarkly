package launchdarkly

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	ldapi "github.com/launchdarkly/api-client-go/v24"
	"github.com/stretchr/testify/assert"
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
	urls = [{
	  kind = "substring"
	  substring = "foo"
	}, {
		kind = "regex"
		pattern = "foo"
	  }]
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
	urls = [{
	  kind = "substring"
	  substring = "bar"
	}, {
		kind = "regex"
		pattern = "bar"
	  }]
}
`

	testAccMetricCustomWithAnalysisUnitsFmt = `
resource "launchdarkly_metric" "custom" {
	project_key = "%s"
	key         = "custom-metric"
	name        = "Custom Metric"
	event_key   = "Custom event"
	kind        = "custom"
	is_numeric  = false

	analysis_units = [
		"request",
		"user"
	]
}
`

	testAccMetricCustomWithAnalysisUnitsUpdateFmt = `
resource "launchdarkly_metric" "custom" {
	project_key = "%s"
	key         = "custom-metric"
	name        = "Custom Metric"
	event_key   = "Custom event"
	kind        = "custom"
	is_numeric  = false

	analysis_units = [
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
			defaultRandomizationUnit := *ldapi.NewRandomizationUnitInput(randomizationUnit)
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
		randomizationUnitsInput = append(randomizationUnitsInput, *ldapi.NewRandomizationUnitInput(randomizationUnit))
	}

	// Update the project's experimentation settings to make the new context available for experiments
	expSettings := ldapi.RandomizationSettingsPut{
		RandomizationUnits: randomizationUnitsInput,
	}
	_, _, err = client.ld.ExperimentsApi.PutExperimentationSettings(betaClient.ctx, projectKey).RandomizationSettingsPut(expSettings).Execute()
	return err
}

func TestAccMetric_BasicCreateAndUpdate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_metric.basic"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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

func TestAccMetric_WithAnalysisUnits(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_metric.custom"

	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)
	// In order to add additional randomization units we need to update the project's context kind and
	// experimentation settings. Because this can only be done using beta endpoints we can't set this up via Terraform.
	betaClient, err := newBetaClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
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
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccMetricCustomWithAnalysisUnitsFmt, projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMetricExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Custom Metric"),
					resource.TestCheckResourceAttr(resourceName, KEY, "custom-metric"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, KIND, "custom"),
					resource.TestCheckResourceAttr(resourceName, EVENT_KEY, "Custom event"),
					resource.TestCheckResourceAttr(resourceName, IS_NUMERIC, "false"),
					resource.TestCheckResourceAttr(resourceName, ANALYSIS_UNITS+".0", "request"),
					resource.TestCheckResourceAttr(resourceName, ANALYSIS_UNITS+".1", "user"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccMetricCustomWithAnalysisUnitsUpdateFmt, projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMetricExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Custom Metric"),
					resource.TestCheckResourceAttr(resourceName, KEY, "custom-metric"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, KIND, "custom"),
					resource.TestCheckResourceAttr(resourceName, EVENT_KEY, "Custom event"),
					resource.TestCheckResourceAttr(resourceName, IS_NUMERIC, "false"),
					resource.TestCheckResourceAttr(resourceName, ANALYSIS_UNITS+".0", "organization"),
					resource.TestCheckResourceAttr(resourceName, ANALYSIS_UNITS+".1", "request"),
					resource.TestCheckResourceAttr(resourceName, ANALYSIS_UNITS+".2", "user"),
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
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
		client := mustTestAccClient()
		_, _, err := client.ld.MetricsApi.GetMetric(client.ctx, projKey, metricKey).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting metric. %s", err)
		}
		return nil
	}
}

const archiveRequiredMessage = "You must archive the metric before you can delete it."

const metricInUseMessage = "Metric is still in use in the following experiments: checkout-test"

func writeConflict(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusConflict)
	mustWrite(w, []byte(fmt.Sprintf(`{"code":"conflict","message":%q}`, message)))
}

func writeMetric(w http.ResponseWriter, archived bool) {
	w.Header().Set("Content-Type", "application/json")
	mustWriteJSON(w, map[string]interface{}{
		"_id":           "5f1e2c8a9d4b3a0011223344",
		"_versionId":    "b9841523-e970-4db9-a747-9da740cb6ec4",
		"key":           "my-metric",
		"name":          "My metric",
		"kind":          "custom",
		"_links":        map[string]interface{}{},
		"tags":          []string{},
		"_creationDate": 1786553140881,
		"dataSource":    map[string]interface{}{"key": "launchdarkly-hosted"},
		"archived":      archived,
	})
}

func decodePatchValue(t *testing.T, r *http.Request, path string) interface{} {
	t.Helper()
	var ops []struct {
		Op    string      `json:"op"`
		Path  string      `json:"path"`
		Value interface{} `json:"value"`
	}
	require.NoError(t, json.NewDecoder(r.Body).Decode(&ops))
	for _, op := range ops {
		if op.Path == path {
			return op.Value
		}
	}
	t.Fatalf("no patch operation for %q in %+v", path, ops)
	return nil
}

func TestArchiveAndDeleteMetric(t *testing.T) {
	t.Run("archives before deleting", func(t *testing.T) {
		var requests []string
		var patched []interface{}
		client, ts := createTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
			requests = append(requests, r.Method+" "+r.URL.Path)
			switch r.Method {
			case http.MethodGet:
				writeMetric(w, false)
			case http.MethodPatch:
				patched = append(patched, decodePatchValue(t, r, "/archived"))
				writeMetric(w, true)
			default:
				w.WriteHeader(http.StatusNoContent)
			}
		})
		defer ts.Close()

		err := (&MetricResource{client: client}).archiveAndDeleteMetric("my-project", "my-metric")

		require.NoError(t, err)
		assert.Equal(t, []string{
			"GET /api/v2/metrics/my-project/my-metric",
			"PATCH /api/v2/metrics/my-project/my-metric",
			"DELETE /api/v2/metrics/my-project/my-metric",
		}, requests)
		assert.Equal(t, []interface{}{true}, patched)
	})

	t.Run("satisfies an API that refuses to delete an unarchived metric", func(t *testing.T) {
		archived := false
		client, ts := createTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				writeMetric(w, archived)
			case http.MethodPatch:
				archived = decodePatchValue(t, r, "/archived") == true
				writeMetric(w, archived)
			case http.MethodDelete:
				if !archived {
					writeConflict(w, archiveRequiredMessage)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}
		})
		defer ts.Close()

		err := (&MetricResource{client: client}).archiveAndDeleteMetric("my-project", "my-metric")

		require.NoError(t, err)
	})

	t.Run("does not re-archive a metric that is already archived", func(t *testing.T) {
		var requests []string
		client, ts := createTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
			requests = append(requests, r.Method)
			if r.Method == http.MethodGet {
				writeMetric(w, true)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})
		defer ts.Close()

		err := (&MetricResource{client: client}).archiveAndDeleteMetric("my-project", "my-metric")

		require.NoError(t, err)
		assert.Equal(t, []string{http.MethodGet, http.MethodDelete}, requests)
	})

	t.Run("restores the archive when the delete fails", func(t *testing.T) {
		var patched []interface{}
		client, ts := createTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				writeMetric(w, false)
			case http.MethodPatch:
				patched = append(patched, decodePatchValue(t, r, "/archived"))
				writeMetric(w, true)
			case http.MethodDelete:
				writeConflict(w, metricInUseMessage)
			}
		})
		defer ts.Close()

		err := (&MetricResource{client: client}).archiveAndDeleteMetric("my-project", "my-metric")

		require.Error(t, err)
		assert.Equal(t, []interface{}{true, false}, patched)
		assert.Contains(t, handleLdapiErr(err).Error(), metricInUseMessage)
	})

	t.Run("leaves an already-archived metric archived when the delete fails", func(t *testing.T) {
		var requests []string
		client, ts := createTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
			requests = append(requests, r.Method)
			if r.Method == http.MethodGet {
				writeMetric(w, true)
				return
			}
			writeConflict(w, metricInUseMessage)
		})
		defer ts.Close()

		err := (&MetricResource{client: client}).archiveAndDeleteMetric("my-project", "my-metric")

		require.Error(t, err)
		assert.Equal(t, []string{http.MethodGet, http.MethodDelete}, requests)
		assert.Contains(t, handleLdapiErr(err).Error(), metricInUseMessage)
	})

	t.Run("reports the blocking dependency when archiving is refused", func(t *testing.T) {
		var requests []string
		client, ts := createTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
			requests = append(requests, r.Method)
			if r.Method == http.MethodGet {
				writeMetric(w, false)
				return
			}
			writeConflict(w, metricInUseMessage)
		})
		defer ts.Close()

		err := (&MetricResource{client: client}).archiveAndDeleteMetric("my-project", "my-metric")

		require.Error(t, err)
		assert.Equal(t, []string{http.MethodGet, http.MethodPatch}, requests)
		assert.Contains(t, handleLdapiErr(err).Error(), metricInUseMessage)
	})

	t.Run("reports that the archived state could not be read", func(t *testing.T) {
		var requests []string
		client, ts := createTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
			requests = append(requests, r.Method)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			mustWrite(w, []byte(`{"code":"internal_error","message":"metric read unavailable"}`))
		})
		defer ts.Close()

		err := (&MetricResource{client: client}).archiveAndDeleteMetric("my-project", "my-metric")

		require.Error(t, err)
		assert.Equal(t, []string{http.MethodGet}, requests)
		assert.Contains(t, handleLdapiErr(err).Error(), "metric read unavailable")
	})

	t.Run("reports both failures when restoring the archive also fails", func(t *testing.T) {
		deletes := 0
		client, ts := createTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				writeMetric(w, false)
			case http.MethodPatch:
				if decodePatchValue(t, r, "/archived") == false {
					writeConflict(w, "cannot unarchive metric")
					return
				}
				writeMetric(w, true)
			case http.MethodDelete:
				deletes++
				writeConflict(w, metricInUseMessage)
			}
		})
		defer ts.Close()

		err := (&MetricResource{client: client}).archiveAndDeleteMetric("my-project", "my-metric")

		require.Error(t, err)
		assert.Equal(t, 1, deletes)
		assert.Contains(t, err.Error(), metricInUseMessage)
		assert.Contains(t, err.Error(), "metric left archived, restoring it failed")
	})
}
