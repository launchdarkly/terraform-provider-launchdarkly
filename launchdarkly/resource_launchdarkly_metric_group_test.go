package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// The metric group config depends on two custom metrics existing in the
// project. We serialize their creation with depends_on to avoid tripping the
// account-wide API rate limit, then reference their keys from the group.
const testAccMetricGroupStandardFmt = `
resource "launchdarkly_metric" "one" {
	project_key = launchdarkly_project.test.key
	key         = "mg-metric-one"
	name        = "MG Metric One"
	kind        = "custom"
	event_key   = "event-one"
	is_numeric  = false
}

resource "launchdarkly_metric" "two" {
	project_key = launchdarkly_project.test.key
	key         = "mg-metric-two"
	name        = "MG Metric Two"
	kind        = "custom"
	event_key   = "event-two"
	is_numeric  = false

	depends_on = [launchdarkly_metric.one]
}

resource "launchdarkly_metric_group" "test" {
	project_key   = launchdarkly_project.test.key
	key           = "checkout-group"
	name          = "Checkout group"
	kind          = "standard"
	description   = "A standard metric group"
	maintainer_id = "%s"

	metrics = [
		{ key = launchdarkly_metric.one.key },
	]

	tags = ["test"]
}
`

const testAccMetricGroupStandardUpdateFmt = `
resource "launchdarkly_metric" "one" {
	project_key = launchdarkly_project.test.key
	key         = "mg-metric-one"
	name        = "MG Metric One"
	kind        = "custom"
	event_key   = "event-one"
	is_numeric  = false
}

resource "launchdarkly_metric" "two" {
	project_key = launchdarkly_project.test.key
	key         = "mg-metric-two"
	name        = "MG Metric Two"
	kind        = "custom"
	event_key   = "event-two"
	is_numeric  = false

	depends_on = [launchdarkly_metric.one]
}

resource "launchdarkly_metric_group" "test" {
	project_key   = launchdarkly_project.test.key
	key           = "checkout-group"
	name          = "Checkout group updated"
	kind          = "standard"
	description   = "An updated standard metric group"
	maintainer_id = "%s"

	metrics = [
		{ key = launchdarkly_metric.one.key },
		{ key = launchdarkly_metric.two.key },
	]

	tags = ["test", "updated"]
}
`

const testAccMetricGroupFunnelFmt = `
resource "launchdarkly_metric" "one" {
	project_key = launchdarkly_project.test.key
	key         = "mg-metric-one"
	name        = "MG Metric One"
	kind        = "custom"
	event_key   = "event-one"
	is_numeric  = false
}

resource "launchdarkly_metric" "two" {
	project_key = launchdarkly_project.test.key
	key         = "mg-metric-two"
	name        = "MG Metric Two"
	kind        = "custom"
	event_key   = "event-two"
	is_numeric  = false

	depends_on = [launchdarkly_metric.one]
}

resource "launchdarkly_metric_group" "test" {
	project_key   = launchdarkly_project.test.key
	key           = "funnel-group"
	name          = "Funnel group"
	kind          = "funnel"
	maintainer_id = "%s"

	metrics = [
		{
			key           = launchdarkly_metric.one.key
			name_in_group = "Step one"
		},
		{
			key           = launchdarkly_metric.two.key
			name_in_group = "Step two"
		},
	]
}
`

func TestAccMetricGroup_Standard(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	maintainerID := firstMemberIDForTest(t)
	resourceName := "launchdarkly_metric_group.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMetricGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, fmt.Sprintf(testAccMetricGroupStandardFmt, maintainerID)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckMetricGroupExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Checkout group"),
					resource.TestCheckResourceAttr(resourceName, KEY, "checkout-group"),
					resource.TestCheckResourceAttr(resourceName, KIND, "standard"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, MAINTAINER_ID, maintainerID),
					resource.TestCheckResourceAttr(resourceName, "metrics.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "metrics.0.key", "mg-metric-one"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withRandomProject(projectKey, fmt.Sprintf(testAccMetricGroupStandardUpdateFmt, maintainerID)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMetricGroupExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Checkout group updated"),
					resource.TestCheckResourceAttr(resourceName, "metrics.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "metrics.1.key", "mg-metric-two"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
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

func TestAccMetricGroup_Funnel(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	maintainerID := firstMemberIDForTest(t)
	resourceName := "launchdarkly_metric_group.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMetricGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, fmt.Sprintf(testAccMetricGroupFunnelFmt, maintainerID)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckMetricGroupExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KIND, "funnel"),
					resource.TestCheckResourceAttr(resourceName, "metrics.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "metrics.0.name_in_group", "Step one"),
					resource.TestCheckResourceAttr(resourceName, "metrics.1.name_in_group", "Step two"),
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

func testAccCheckMetricGroupExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		key, ok := rs.Primary.Attributes[KEY]
		if !ok {
			return fmt.Errorf("metric group key not found: %s", resourceName)
		}
		projKey, ok := rs.Primary.Attributes[PROJECT_KEY]
		if !ok {
			return fmt.Errorf("project key not found: %s", resourceName)
		}
		beta, err := newMetricGroupBetaClient(mustTestAccClient())
		if err != nil {
			return err
		}
		_, _, err = beta.ld.MetricsBetaApi.GetMetricGroup(beta.ctx, projKey, key).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting metric group: %s", err)
		}
		return nil
	}
}

func testAccCheckMetricGroupDestroy(s *terraform.State) error {
	beta, err := newMetricGroupBetaClient(mustTestAccClient())
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_metric_group" {
			continue
		}
		projKey := rs.Primary.Attributes[PROJECT_KEY]
		key := rs.Primary.Attributes[KEY]
		_, res, err := beta.ld.MetricsBetaApi.GetMetricGroup(beta.ctx, projKey, key).Execute()
		if isStatusNotFound(res) {
			continue
		}
		if err == nil {
			return fmt.Errorf("metric group %q still exists in project %q", key, projKey)
		}
	}
	return nil
}
