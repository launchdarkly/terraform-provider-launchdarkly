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
	testAccDataSourceReleasePolicyBasic = `
data "launchdarkly_release_policy" "test" {
	project_key = "%s"
	key         = "%s"
}
`
)

func TestAccDataSourceReleasePolicy_noMatchReturnsError(t *testing.T) {
	projectKey := "nonexistent-project-key"
	policyKey := "nonexistent-policy-key"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccDataSourceReleasePolicyBasic, projectKey, policyKey),
				ExpectError: regexp.MustCompile(`failed to get release policy`),
			},
		},
	})
}

func TestAccDataSourceReleasePolicy_existsGuarded(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectName := "release-policy-data-source-test-" + projectKey
	policyKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	policyName := "Test Release Policy"

	resourceName := "data.launchdarkly_release_policy.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "launchdarkly_project" "test" {
	name = "%s"
	key  = "%s"
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

resource "launchdarkly_release_policy" "test" {
	project_key    = launchdarkly_project.test.key
	key            = "%s"
	name           = "%s"
	release_method = "guarded-release"

	scope {
		environment_keys = ["test-env", "production"]
	}

	guarded_release_config {
		rollback_on_regression = true
		min_sample_size        = 100
	}
}

data "launchdarkly_release_policy" "test" {
	project_key = launchdarkly_project.test.key
	key         = launchdarkly_release_policy.test.key
}
`, projectName, projectKey, policyKey, policyName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, PROJECT_KEY),
					resource.TestCheckResourceAttrSet(resourceName, KEY),
					resource.TestCheckResourceAttrSet(resourceName, NAME),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, KEY, policyKey),
					resource.TestCheckResourceAttr(resourceName, NAME, policyName),
					resource.TestCheckResourceAttr(resourceName, RELEASE_METHOD, "guarded-release"),
					resource.TestCheckResourceAttrSet(resourceName, ID),
					resource.TestCheckResourceAttr(resourceName, "scope.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "scope.0.environment_keys.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "scope.0.environment_keys.0", "test-env"),
					resource.TestCheckResourceAttr(resourceName, "scope.0.environment_keys.1", "production"),
					resource.TestCheckResourceAttr(resourceName, "guarded_release_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "guarded_release_config.0.rollback_on_regression", "true"),
					resource.TestCheckResourceAttr(resourceName, "guarded_release_config.0.min_sample_size", "100"),
				),
			},
		},
	})
}
