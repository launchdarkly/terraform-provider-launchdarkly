package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go/v7"
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

func testAccDataSourceMetricScaffold(client *Client, projectKey string, metricBody ldapi.MetricPost) (*ldapi.MetricRep, error) {
	projectBody := ldapi.ProjectPost{
		Name: "Metric Test Project",
		Key:  projectKey,
	}
	project, err := testAccDataSourceProjectCreate(client, projectBody)
	if err != nil {
		return nil, err
	}

	metric, _, err := client.ld.MetricsApi.PostMetric(client.ctx, project.Key).MetricPost(metricBody).Execute()
	if err != nil {
		return nil, err
	}

	return &metric, nil
}

func TestAccDataSourceMetric_noMatchReturnsError(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false)
	require.NoError(t, err)
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectBody := ldapi.ProjectPost{
		Name: "Terraform Metric Test Project",
		Key:  projectKey,
	}
	project, err := testAccDataSourceProjectCreate(client, projectBody)
	require.NoError(t, err)

	defer func() {
		err := testAccDataSourceProjectDelete(client, projectKey)
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
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false)
	require.NoError(t, err)

	metricName := "Metric Data Source Test"
	metricKey := "metric-ds-testing"
	metricUrlKind := "substring"
	metricUrlSubstring := "foo"
	metricBody := ldapi.MetricPost{
		Name: &metricName,
		Key:  metricKey,
		Kind: "pageview",
		Urls: &[]ldapi.UrlPost{{
			Kind:      &metricUrlKind,
			Substring: &metricUrlSubstring,
		}},
		Description: ldapi.PtrString("a metric to test the terraform metric data source"),
	}
	metric, err := testAccDataSourceMetricScaffold(client, projectKey, metricBody)
	require.NoError(t, err)

	defer func() {
		err := testAccDataSourceProjectDelete(client, projectKey)
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
					resource.TestCheckResourceAttr(resourceName, "urls.0.kind", metricUrlKind),
					resource.TestCheckResourceAttr(resourceName, "urls.0.substring", metricUrlSubstring),
				),
			},
		},
	})
}
