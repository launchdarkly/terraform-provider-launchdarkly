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
	testAccReleasePolicyGuardedCreate = `
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

resource "launchdarkly_release_policy" "test" {
	project_key    = launchdarkly_project.test.key
	key            = "%s"
	name           = "%s"
	release_method = "guarded-release"

	scope {
		environment_keys = ["test-env", "production"]
	}

	guarded_release_config {
		rollback_on_regression = %t
		min_sample_size        = %d
	}
}
`

	testAccReleasePolicyGuardedUpdate = `
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

resource "launchdarkly_release_policy" "test" {
	project_key    = launchdarkly_project.test.key
	key            = "%s"
	name           = "%s"
	release_method = "guarded-release"

	scope {
		environment_keys = ["production"]
	}

	guarded_release_config {
		rollback_on_regression = %t
		min_sample_size        = %d
	}
}
`

	testAccReleasePolicyProgressiveCreate = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "Test project"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_release_policy" "test" {
	project_key    = launchdarkly_project.test.key
	key            = "%s"
	name           = "%s"
	release_method = "progressive-release"

	scope {
		environment_keys = ["test-env"]
	}
}
`

	testAccReleasePolicyMinimal = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "Test project"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_release_policy" "test" {
	project_key    = launchdarkly_project.test.key
	key            = "%s"
	name           = "%s"
	release_method = "guarded-release"

	guarded_release_config {
		rollback_on_regression = true
	}
}
`

	testAccReleasePolicyGuardedWithoutMinSampleSize = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "Test project"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_release_policy" "test" {
	project_key    = launchdarkly_project.test.key
	key            = "%s"
	name           = "%s"
	release_method = "guarded-release"

	scope {
		environment_keys = ["test-env"]
	}

	guarded_release_config {
		rollback_on_regression = %t
	}
}
`

	testAccReleasePolicyInvalidReleaseMethod = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "Test project"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_release_policy" "test" {
	project_key    = launchdarkly_project.test.key
	key            = "%s"
	name           = "Test Release Policy"
	release_method = "invalid-method"

	guarded_release_config {
		rollback_on_regression = true
	}
}
`

	testAccReleasePolicyInvalidMinSampleSize = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "Test project"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_release_policy" "test" {
	project_key    = launchdarkly_project.test.key
	key            = "%s"
	name           = "Test Release Policy"
	release_method = "guarded-release"

	guarded_release_config {
		rollback_on_regression = true
		min_sample_size        = 3
	}
}
`
)

func TestAccReleasePolicy_GuardedRelease(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	policyKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	policyName := "Test Guarded Release Policy"
	policyNameUpdated := "Updated Guarded Release Policy"
	resourceName := "launchdarkly_release_policy.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckReleasePolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccReleasePolicyGuardedCreate, projectKey, policyKey, policyName, true, 100),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckReleasePolicyExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, KEY, policyKey),
					resource.TestCheckResourceAttr(resourceName, NAME, policyName),
					resource.TestCheckResourceAttr(resourceName, RELEASE_METHOD, "guarded-release"),
					resource.TestCheckResourceAttr(resourceName, "scope.0.environment_keys.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "scope.0.environment_keys.0", "test-env"),
					resource.TestCheckResourceAttr(resourceName, "scope.0.environment_keys.1", "production"),
					resource.TestCheckResourceAttr(resourceName, "guarded_release_config.0.rollback_on_regression", "true"),
					resource.TestCheckResourceAttr(resourceName, "guarded_release_config.0.min_sample_size", "100"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccReleasePolicyGuardedUpdate, projectKey, policyKey, policyNameUpdated, false, 200),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckReleasePolicyExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, KEY, policyKey),
					resource.TestCheckResourceAttr(resourceName, NAME, policyNameUpdated),
					resource.TestCheckResourceAttr(resourceName, RELEASE_METHOD, "guarded-release"),
					resource.TestCheckResourceAttr(resourceName, "scope.0.environment_keys.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "scope.0.environment_keys.0", "production"),
					resource.TestCheckResourceAttr(resourceName, "guarded_release_config.0.rollback_on_regression", "false"),
					resource.TestCheckResourceAttr(resourceName, "guarded_release_config.0.min_sample_size", "200"),
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

func TestAccReleasePolicy_ProgressiveRelease(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	policyKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	policyName := "Test Progressive Release Policy"
	resourceName := "launchdarkly_release_policy.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckReleasePolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccReleasePolicyProgressiveCreate, projectKey, policyKey, policyName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckReleasePolicyExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, KEY, policyKey),
					resource.TestCheckResourceAttr(resourceName, NAME, policyName),
					resource.TestCheckResourceAttr(resourceName, RELEASE_METHOD, "progressive-release"),
					resource.TestCheckResourceAttr(resourceName, "scope.0.environment_keys.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "scope.0.environment_keys.0", "test-env"),
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

func TestAccReleasePolicy_GuardedWithoutMinSampleSize(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	policyKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	policyName := "Guarded Release Without Min Sample Size"
	resourceName := "launchdarkly_release_policy.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckReleasePolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccReleasePolicyGuardedWithoutMinSampleSize, projectKey, policyKey, policyName, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckReleasePolicyExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, KEY, policyKey),
					resource.TestCheckResourceAttr(resourceName, NAME, policyName),
					resource.TestCheckResourceAttr(resourceName, RELEASE_METHOD, "guarded-release"),
					resource.TestCheckResourceAttr(resourceName, "scope.0.environment_keys.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "scope.0.environment_keys.0", "test-env"),
					resource.TestCheckResourceAttr(resourceName, "guarded_release_config.0.rollback_on_regression", "true"),
					// min_sample_size should not be set in the configuration, so it should use the default (0)
					resource.TestCheckResourceAttr(resourceName, "guarded_release_config.0.min_sample_size", "0"),
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

func TestAccReleasePolicy_Import(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	policyKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	policyName := "Import Test Release Policy"
	resourceName := "launchdarkly_release_policy.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckReleasePolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccReleasePolicyMinimal, projectKey, policyKey, policyName),
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
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes[PROJECT_KEY], rs.Primary.Attributes[KEY]), nil
				},
			},
		},
	})
}

func TestAccReleasePolicy_InvalidReleaseMethod(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	policyKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckReleasePolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccReleasePolicyInvalidReleaseMethod, projectKey, policyKey),
				ExpectError: regexp.MustCompile(`expected release_method to be one of \["guarded-release" "progressive-release"\], got invalid-method`),
			},
		},
	})
}

func TestAccReleasePolicy_InvalidMinSampleSize(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	policyKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckReleasePolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccReleasePolicyInvalidMinSampleSize, projectKey, policyKey),
				ExpectError: regexp.MustCompile(`expected min_sample_size to be at least \(5\), got 3`),
			},
		},
	})
}

func TestAccReleasePolicy_InvalidKey(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	invalidPolicyKey := "invalid key with spaces"
	policyName := "Test Release Policy"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckReleasePolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccReleasePolicyMinimal, projectKey, invalidPolicyKey, policyName),
				ExpectError: regexp.MustCompile(`Must contain only letters, numbers, '.', '-', or '_' and must start with an alphanumeric`),
			},
		},
	})
}

func testAccCheckReleasePolicyExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("release policy ID is not set")
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		policyKey := rs.Primary.Attributes[KEY]

		client := testAccProvider.Meta().(*Client)
		_, _, err := getReleasePolicy(client, projectKey, policyKey)
		if err != nil {
			return fmt.Errorf("received an error getting release policy. %s", err)
		}
		return nil
	}
}

func testAccCheckReleasePolicyDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_release_policy" {
			continue
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		policyKey := rs.Primary.Attributes[KEY]

		_, res, err := getReleasePolicyRaw(client, projectKey, policyKey)
		if isStatusNotFound(res) {
			continue
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("release policy still exists")
	}
	return nil
}
