package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go/v22"
	"github.com/stretchr/testify/require"
)

const testAccDataSourceMetricGroup = `
data "launchdarkly_metric_group" "testing" {
	key         = "%s"
	project_key = "%s"
}
`

// testAccDataSourceMetricGroupScaffold creates a project, a custom metric, and
// a standard metric group referencing that metric, all via the API. The metric
// groups endpoints are beta, so the group is created with the beta client.
func testAccDataSourceMetricGroupScaffold(client *Client, beta *Client, projectKey, maintainerID string) (*ldapi.MetricGroupRep, error) {
	projectBody := ldapi.ProjectPost{
		Name: "Metric Group Test Project",
		Key:  projectKey,
	}
	project, err := testAccProjectScaffoldCreate(client, projectBody)
	if err != nil {
		return nil, err
	}

	metricName := "MG DS Metric"
	metricKey := "mg-ds-metric"
	eventKey := "mg-ds-event"
	isNumeric := false
	metricBody := ldapi.MetricPost{
		Name:      &metricName,
		Key:       metricKey,
		Kind:      "custom",
		EventKey:  &eventKey,
		IsNumeric: &isNumeric,
	}
	if _, _, err := client.ld.MetricsApi.PostMetric(client.ctx, project.Key).MetricPost(metricBody).Execute(); err != nil {
		return nil, err
	}

	groupKey := "mg-ds-group"
	post := ldapi.MetricGroupPost{
		Key:          &groupKey,
		Name:         "MG DS Group",
		Kind:         "standard",
		MaintainerId: maintainerID,
		Tags:         []string{"test"},
		Metrics: []ldapi.MetricInMetricGroupInput{
			{Key: metricKey, NameInGroup: ""},
		},
		Description: ldapi.PtrString("a metric group to test the terraform data source"),
	}
	group, _, err := beta.ld.MetricsBetaApi.CreateMetricGroup(beta.ctx, project.Key).MetricGroupPost(post).Execute()
	if err != nil {
		return nil, err
	}
	return group, nil
}

func TestAccDataSourceMetricGroup_noMatchReturnsError(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectBody := ldapi.ProjectPost{
		Name: "Terraform Metric Group Test Project",
		Key:  projectKey,
	}
	project, err := testAccProjectScaffoldCreate(client, projectBody)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, testAccProjectScaffoldDelete(client, projectKey))
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccDataSourceMetricGroup, "nonexistent-group", project.Key),
				ExpectError: regexp.MustCompile("Error: 404 Not Found"),
			},
		},
	})
}

func TestAccDataSourceMetricGroup_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	maintainerID := firstMemberIDForTest(t)
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)
	beta, err := newMetricGroupBetaClient(client)
	require.NoError(t, err)

	group, err := testAccDataSourceMetricGroupScaffold(client, beta, projectKey, maintainerID)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, testAccProjectScaffoldDelete(client, projectKey))
	}()

	resourceName := "data.launchdarkly_metric_group.testing"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceMetricGroup, group.Key, projectKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, KEY),
					resource.TestCheckResourceAttrSet(resourceName, NAME),
					resource.TestCheckResourceAttr(resourceName, KEY, group.Key),
					resource.TestCheckResourceAttr(resourceName, NAME, group.Name),
					resource.TestCheckResourceAttr(resourceName, KIND, group.Kind),
					resource.TestCheckResourceAttr(resourceName, ID, projectKey+"/"+group.Key),
					resource.TestCheckResourceAttr(resourceName, "metrics.0.key", group.Metrics[0].Key),
				),
			},
		},
	})
}
