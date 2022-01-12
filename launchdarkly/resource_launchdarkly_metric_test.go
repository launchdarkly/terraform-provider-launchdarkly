package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
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
)

func TestAccMetric_Basic(t *testing.T) {
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
		},
	})
}

func TestAccMetric_Update(t *testing.T) {
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
