package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	testAccDataSourceReleasePolicyOrderConfig = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "Test project for release policy order datasource"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
	environments {
		name  = "Production Environment"
		key   = "production"
		color = "ff0000"
	}
}

resource "launchdarkly_release_policy" "policy1" {
	project_key    = launchdarkly_project.test.key
	key            = "policy-1"
	name           = "Policy 1"
	release_method = "guarded-release"

	scope {
		environment_keys = ["test-env", "production"]
	}
	
	guarded_release_config {
		rollback_on_regression = true
	}
}

resource "launchdarkly_release_policy" "policy2" {
	project_key    = launchdarkly_project.test.key
	key            = "policy-2"
	name           = "Policy 2"
	release_method = "guarded-release"

	scope {
		environment_keys = ["test-env", "production"]
	}
	
	guarded_release_config {
		rollback_on_regression = true
	}
}

resource "launchdarkly_release_policy_order" "test" {
	project_key = launchdarkly_project.test.key
	release_policy_keys = [
		launchdarkly_release_policy.policy1.key,
		launchdarkly_release_policy.policy2.key,
	]
}

data "launchdarkly_release_policy_order" "test" {
	project_key = launchdarkly_project.test.key
	depends_on = [launchdarkly_release_policy_order.test]
}
`
)

func TestAccDataSourceReleasePolicyOrder_noMatchReturnsError(t *testing.T) {
	projectKey := "nonexistent-project-key"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(`data "launchdarkly_release_policy_order" "test" { project_key = "%s" }`, projectKey),
				ExpectError: regexp.MustCompile(`failed to get release policy order for project "nonexistent-project-key"`),
			},
		},
	})
}

func TestAccDataSourceReleasePolicyOrder_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "data.launchdarkly_release_policy_order.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceReleasePolicyOrderConfig, projectKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, PROJECT_KEY),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, "release_policy_keys.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "release_policy_keys.0", "policy-1"),
					resource.TestCheckResourceAttr(resourceName, "release_policy_keys.1", "policy-2"),
				),
			},
		},
	})
}
