package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	testAccReleasePolicyOrderCreate = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "Test project"
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

resource "launchdarkly_release_policy" "policy3" {
	project_key    = launchdarkly_project.test.key
	key            = "policy-3"
	name           = "Policy 3"
	release_method = "progressive-release"

	scope {
		environment_keys = ["test-env", "production"]
	}
}

resource "launchdarkly_release_policy_order" "test" {
	project_key = launchdarkly_project.test.key
	release_policy_keys = [
		launchdarkly_release_policy.policy1.key,
		launchdarkly_release_policy.policy2.key,
		launchdarkly_release_policy.policy3.key
	]
}
`

	testAccReleasePolicyOrderUpdate = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "Test project"
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

resource "launchdarkly_release_policy" "policy3" {
	project_key    = launchdarkly_project.test.key
	key            = "policy-3"
	name           = "Policy 3"
	release_method = "progressive-release"

	scope {
		environment_keys = ["test-env", "production"]
	}
}

resource "launchdarkly_release_policy_order" "test" {
	project_key = launchdarkly_project.test.key
	release_policy_keys = [
		launchdarkly_release_policy.policy3.key,
		launchdarkly_release_policy.policy1.key,
		launchdarkly_release_policy.policy2.key
	]
}
`

	testAccReleasePolicyOrderWithDefaultPolicy = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "Test project"
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

resource "launchdarkly_release_policy" "default_policy" {
	project_key    = launchdarkly_project.test.key
	key            = "default-policy"
	name           = "Default Policy"
	release_method = "progressive-release"
	# No scope - this is a default policy
}

resource "launchdarkly_release_policy_order" "test" {
	project_key = launchdarkly_project.test.key
	release_policy_keys = [
		launchdarkly_release_policy.policy1.key,
		launchdarkly_release_policy.default_policy.key
	]
}
`

	testAccReleasePolicyOrderMissingPolicies = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "Test project"
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

resource "launchdarkly_release_policy" "policy3" {
	project_key    = launchdarkly_project.test.key
	key            = "policy-3"
	name           = "Policy 3"
	release_method = "progressive-release"

	scope {
		environment_keys = ["test-env", "production"]
	}
}

resource "launchdarkly_release_policy_order" "test" {
	project_key = launchdarkly_project.test.key
	# Only include policy1 and policy2, intentionally omitting policy3
	release_policy_keys = [
		launchdarkly_release_policy.policy1.key,
		launchdarkly_release_policy.policy2.key
	]
}
`
)

func TestAccReleasePolicyOrder_Basic(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_release_policy_order.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccReleasePolicyOrderCreate, projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckReleasePolicyOrderExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, "release_policy_keys.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "release_policy_keys.0", "policy-1"),
					resource.TestCheckResourceAttr(resourceName, "release_policy_keys.1", "policy-2"),
					resource.TestCheckResourceAttr(resourceName, "release_policy_keys.2", "policy-3"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccReleasePolicyOrderUpdate, projectKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckReleasePolicyOrderExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, "release_policy_keys.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "release_policy_keys.0", "policy-3"),
					resource.TestCheckResourceAttr(resourceName, "release_policy_keys.1", "policy-1"),
					resource.TestCheckResourceAttr(resourceName, "release_policy_keys.2", "policy-2"),
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

func TestAccReleasePolicyOrder_Import(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_release_policy_order.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccReleasePolicyOrderCreate, projectKey),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources[resourceName]
					if !ok {
						return "", fmt.Errorf("not found: %s", resourceName)
					}
					return rs.Primary.Attributes[PROJECT_KEY], nil
				},
			},
		},
	})
}

func TestAccReleasePolicyOrder_WithDefaultPolicy(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccReleasePolicyOrderWithDefaultPolicy, projectKey),
				ExpectError: regexp.MustCompile("Status: 400.*Default policy key is not allowed in ordering"),
			},
		},
	})
}

func TestAccReleasePolicyOrder_MissingPolicies(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccReleasePolicyOrderMissingPolicies, projectKey),
				ExpectError: regexp.MustCompile("Status: 400.*Missing policy key"),
			},
		},
	})
}

func testAccCheckReleasePolicyOrderExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("release policy order ID is not set")
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]

		client := testAccProvider.Meta().(*Client)
		_, err := getReleasePolicyOrder(client, projectKey)
		if err != nil {
			return fmt.Errorf("received an error getting release policy order. %s", err)
		}
		return nil
	}
}
