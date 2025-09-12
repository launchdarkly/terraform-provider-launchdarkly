package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go/v17"
	"github.com/stretchr/testify/require"
)

const (
	testAccDataSourceMetric = `
data "launchdarkly_metric" "testing" {
	key         = "%s"
	project_key = "%s"
}
`
)

func testAccDataSourceMetricScaffold(client *Client, betaClient *Client, projectKey string, metricBody ldapi.MetricPost) (*ldapi.MetricRep, error) {
	projectBody := ldapi.ProjectPost{
		Name: "Metric Test Project",
		Key:  projectKey,
	}
	project, err := testAccProjectScaffoldCreate(client, projectBody)
	if err != nil {
		return nil, err
	}

	randomizationUnitsInput := make([]ldapi.RandomizationUnitInput, 0, len(metricBody.RandomizationUnits))
	for _, randomizationUnit := range metricBody.RandomizationUnits {
		if randomizationUnit == "user" {
			defaultTrue := true
			defaultRandomizationUnit := *ldapi.NewRandomizationUnitInput(randomizationUnit, randomizationUnit)
			defaultRandomizationUnit.Default = &defaultTrue
			randomizationUnitsInput = append(randomizationUnitsInput, defaultRandomizationUnit)
			continue
		}
		// Add the additional context kinds to the project
		contextKindPayload := ldapi.UpsertContextKindPayload{Name: randomizationUnit}
		_, _, err = client.ld.ContextsApi.PutContextKind(client.ctx, project.Key, randomizationUnit).UpsertContextKindPayload(contextKindPayload).Execute()
		if err != nil {
			return nil, err
		}
		randomizationUnitsInput = append(randomizationUnitsInput, *ldapi.NewRandomizationUnitInput(randomizationUnit, randomizationUnit))
	}

	// Update the project's experimentation settings to make the new context available for experiments
	expSettings := ldapi.RandomizationSettingsPut{
		RandomizationUnits: randomizationUnitsInput,
	}
	_, _, err = client.ld.ExperimentsApi.PutExperimentationSettings(betaClient.ctx, projectKey).RandomizationSettingsPut(expSettings).Execute()
	if err != nil {
		return nil, err
	}

	metric, _, err := client.ld.MetricsApi.PostMetric(client.ctx, project.Key).MetricPost(metricBody).Execute()
	if err != nil {
		return nil, err
	}

	return metric, nil
}

func TestAccDataSourceMetric_noMatchReturnsError(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectBody := ldapi.ProjectPost{
		Name: "Terraform Metric Test Project",
		Key:  projectKey,
	}
	project, err := testAccProjectScaffoldCreate(client, projectBody)
	require.NoError(t, err)

	defer func() {
		err := testAccProjectScaffoldDelete(client, projectKey)
		require.NoError(t, err)
	}()

	metricKey := "nonexistent-metric"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccDataSourceMetric, metricKey, project.Key),
				ExpectError: regexp.MustCompile("Error: 404 Not Found"),
			},
		},
	})
}

func TestAccDataSourceMetric_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)
	betaClient, err := newBetaClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)

	metricName := "Metric Data Source Test"
	metricKey := "metric-ds-testing"
	metricUrlKind := "substring"
	metricUrlSubstring := "foo"
	metricBody := ldapi.MetricPost{
		Name: &metricName,
		Key:  metricKey,
		Kind: "pageview",
		Urls: []ldapi.UrlPost{{
			Kind:      &metricUrlKind,
			Substring: &metricUrlSubstring,
		}},
		Description:        ldapi.PtrString("a metric to test the terraform metric data source"),
		RandomizationUnits: []string{"request", "user"},
	}
	metric, err := testAccDataSourceMetricScaffold(client, betaClient, projectKey, metricBody)
	require.NoError(t, err)

	defer func() {
		err := testAccProjectScaffoldDelete(client, projectKey)
		require.NoError(t, err)
	}()

	resourceName := "data.launchdarkly_metric.testing"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceMetric, metricKey, projectKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, KEY),
					resource.TestCheckResourceAttrSet(resourceName, NAME),
					resource.TestCheckResourceAttrSet(resourceName, PROJECT_KEY),
					resource.TestCheckResourceAttr(resourceName, KEY, metric.Key),
					resource.TestCheckResourceAttr(resourceName, NAME, metric.Name),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, *metric.Description),
					resource.TestCheckResourceAttr(resourceName, ID, projectKey+"/"+metric.Key),
					resource.TestCheckResourceAttr(resourceName, KIND, metric.Kind),
					resource.TestCheckResourceAttr(resourceName, RANDOMIZATION_UNITS+".0", metric.RandomizationUnits[0]),
					resource.TestCheckResourceAttr(resourceName, RANDOMIZATION_UNITS+".1", metric.RandomizationUnits[1]),
					resource.TestCheckResourceAttr(resourceName, "urls.0.kind", metricUrlKind),
					resource.TestCheckResourceAttr(resourceName, "urls.0.substring", metricUrlSubstring),
				),
			},
		},
	})
}

func TestAccDataSourceMetric_ArchivedField(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectBody := ldapi.ProjectPost{
		Name: "Terraform Metric Archived Test Project",
		Key:  projectKey,
	}
	project, err := testAccProjectScaffoldCreate(client, projectBody)
	require.NoError(t, err)

	defer func() {
		err := testAccProjectScaffoldDelete(client, projectKey)
		require.NoError(t, err)
	}()

	// Create archived metric (without Archived field since API client doesn't support it yet)
	archivedMetricKey := "archived-metric"
	archivedMetricName := "Archived Test Metric"
	archivedMetricDescription := "Test metric for archived field testing"
	archivedUrlKind := "substring"
	archivedUrlSubstring := "archived-test"
	archivedMetricBody := ldapi.MetricPost{
		Name:        &archivedMetricName,
		Key:         archivedMetricKey,
		Description: ldapi.PtrString(archivedMetricDescription),
		Kind:        "pageview",
		Tags:        []string{"test", "archived"},
		Urls: []ldapi.UrlPost{
			{
				Kind:      &archivedUrlKind,
				Substring: &archivedUrlSubstring,
			},
		},
	}
	// Create archived metric directly without scaffold (to avoid duplicate project creation)
	_, _, err = client.ld.MetricsApi.PostMetric(client.ctx, project.Key).MetricPost(archivedMetricBody).Execute()
	require.NoError(t, err)

	// Create non-archived metric (without Archived field since API client doesn't support it yet)
	nonArchivedMetricKey := "non-archived-metric"
	nonArchivedMetricName := "Non-Archived Test Metric"
	nonArchivedMetricDescription := "Test metric for non-archived field testing"
	nonArchivedUrlKind := "substring"
	nonArchivedUrlSubstring := "non-archived-test"
	nonArchivedMetricBody := ldapi.MetricPost{
		Name:        &nonArchivedMetricName,
		Key:         nonArchivedMetricKey,
		Description: ldapi.PtrString(nonArchivedMetricDescription),
		Kind:        "pageview",
		Tags:        []string{"test", "non-archived"},
		Urls: []ldapi.UrlPost{
			{
				Kind:      &nonArchivedUrlKind,
				Substring: &nonArchivedUrlSubstring,
			},
		},
	}
	// Create non-archived metric directly without scaffold (to avoid duplicate project creation)
	_, _, err = client.ld.MetricsApi.PostMetric(client.ctx, project.Key).MetricPost(nonArchivedMetricBody).Execute()
	require.NoError(t, err)

	// Test data source configurations
	testAccDataSourceMetricArchived := `
data "launchdarkly_metric" "archived" {
	key         = "%s"
	project_key = "%s"
}
`

	testAccDataSourceMetricNonArchived := `
data "launchdarkly_metric" "non_archived" {
	key         = "%s"
	project_key = "%s"
}
`

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			// Test reading archived metric (currently returns false since API client doesn't support Archived field yet)
			{
				Config: fmt.Sprintf(testAccDataSourceMetricArchived, archivedMetricKey, project.Key),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.launchdarkly_metric.archived", KEY, archivedMetricKey),
					resource.TestCheckResourceAttr("data.launchdarkly_metric.archived", NAME, "Archived Test Metric"),
					resource.TestCheckResourceAttr("data.launchdarkly_metric.archived", PROJECT_KEY, project.Key),
					resource.TestCheckResourceAttr("data.launchdarkly_metric.archived", KIND, "pageview"),
					resource.TestCheckResourceAttr("data.launchdarkly_metric.archived", ARCHIVED, "false"), // Default value until API client supports Archived field
					resource.TestCheckResourceAttr("data.launchdarkly_metric.archived", "tags.0", "test"),
					resource.TestCheckResourceAttr("data.launchdarkly_metric.archived", "tags.1", "archived"),
					resource.TestCheckResourceAttr("data.launchdarkly_metric.archived", "urls.0.kind", "substring"),
					resource.TestCheckResourceAttr("data.launchdarkly_metric.archived", "urls.0.substring", "archived-test"),
				),
			},
			// Test reading non-archived metric
			{
				Config: fmt.Sprintf(testAccDataSourceMetricNonArchived, nonArchivedMetricKey, project.Key),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.launchdarkly_metric.non_archived", KEY, nonArchivedMetricKey),
					resource.TestCheckResourceAttr("data.launchdarkly_metric.non_archived", NAME, "Non-Archived Test Metric"),
					resource.TestCheckResourceAttr("data.launchdarkly_metric.non_archived", PROJECT_KEY, project.Key),
					resource.TestCheckResourceAttr("data.launchdarkly_metric.non_archived", KIND, "pageview"),
					resource.TestCheckResourceAttr("data.launchdarkly_metric.non_archived", ARCHIVED, "false"),
					resource.TestCheckResourceAttr("data.launchdarkly_metric.non_archived", "tags.0", "test"),
					resource.TestCheckResourceAttr("data.launchdarkly_metric.non_archived", "tags.1", "non-archived"),
					resource.TestCheckResourceAttr("data.launchdarkly_metric.non_archived", "urls.0.kind", "substring"),
					resource.TestCheckResourceAttr("data.launchdarkly_metric.non_archived", "urls.0.substring", "non-archived-test"),
				),
			},
		},
	})
}
