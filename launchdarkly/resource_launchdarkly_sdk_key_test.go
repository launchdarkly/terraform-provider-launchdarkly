package launchdarkly

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	testAccSdkKeyCreate = `
resource "launchdarkly_sdk_key" "test" {
	project_key     = launchdarkly_project.test.key
	environment_key = "test"
	key             = "tf-test-sdk-key"
	name            = "Terraform test SDK key"
	description     = "Managed by Terraform"
}
`

	testAccSdkKeyUpdate = `
resource "launchdarkly_sdk_key" "test" {
	project_key     = launchdarkly_project.test.key
	environment_key = "test"
	key             = "tf-test-sdk-key"
	name            = "Terraform test SDK key updated"
	description     = "Updated by Terraform"
}
`

	testAccSdkKeyWithExpiry = `
resource "launchdarkly_sdk_key" "test" {
	project_key     = launchdarkly_project.test.key
	environment_key = "test"
	key             = "tf-test-sdk-key-expiry"
	name            = "Terraform expiry SDK key"
	expiry          = %d
}
`

	testAccSdkKeyExpiryRemoved = `
resource "launchdarkly_sdk_key" "test" {
	project_key     = launchdarkly_project.test.key
	environment_key = "test"
	key             = "tf-test-sdk-key-expiry"
	name            = "Terraform expiry SDK key"
}
`

	testAccSdkKeyExpiryRemovedViaReplace = `
resource "launchdarkly_sdk_key" "test" {
	project_key     = launchdarkly_project.test.key
	environment_key = "test"
	key             = "tf-test-sdk-key-fresh"
	name            = "Terraform expiry SDK key"
}
`
)

func TestAccSdkKey_CreateAndUpdate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	resourceName := "launchdarkly_sdk_key.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSdkKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccSdkKeyCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckSdkKeyExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENVIRONMENT_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, KEY, "tf-test-sdk-key"),
					resource.TestCheckResourceAttr(resourceName, NAME, "Terraform test SDK key"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "Managed by Terraform"),
					resource.TestCheckResourceAttr(resourceName, KIND, "sdk"),
					resource.TestCheckResourceAttrSet(resourceName, VALUE),
					resource.TestCheckResourceAttrSet(resourceName, VERSION),
					resource.TestCheckResourceAttr(resourceName, ID, projectKey+"/test/tf-test-sdk-key"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withRandomProject(projectKey, testAccSdkKeyUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSdkKeyExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Terraform test SDK key updated"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "Updated by Terraform"),
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

// TestAccSdkKey_ExpiryCannotBeRemoved asserts that once an expiry is set, a
// config that removes it fails at plan time rather than producing an
// inconsistent apply. The beta patch model cannot emit a null expiry to clear
// it (PATCH expiry=0 is rejected), and a deleted SDK key identifier is
// tombstoned so the resource cannot be recreated at the same key either — so a
// plan-time error (expiryRemovalGuard) is the only safe behavior.
func TestAccSdkKey_ExpiryCannotBeRemoved(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	resourceName := "launchdarkly_sdk_key.test"
	// Expiry must be within 10 years of now; use ~1 year out so the test stays
	// valid over time (a hardcoded epoch would eventually fall in the past).
	expiry := time.Now().Add(365 * 24 * time.Hour).UnixMilli()
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSdkKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, fmt.Sprintf(testAccSdkKeyWithExpiry, expiry)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSdkKeyExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, EXPIRY, fmt.Sprintf("%d", expiry)),
				),
			},
			{
				Config:      withRandomProject(projectKey, testAccSdkKeyExpiryRemoved),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("Cannot remove expiry from an SDK key"),
			},
			{
				// Dropping the expiry while changing `key` is a replacement
				// under a fresh identifier, which the guard must allow.
				Config: withRandomProject(projectKey, testAccSdkKeyExpiryRemovedViaReplace),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSdkKeyExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "tf-test-sdk-key-fresh"),
					resource.TestCheckNoResourceAttr(resourceName, EXPIRY),
				),
			},
		},
	})
}

func testAccCheckSdkKeyExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("SDK key ID is not set")
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		environmentKey := rs.Primary.Attributes[ENVIRONMENT_KEY]
		sdkKeyKey := rs.Primary.Attributes[KEY]

		client := mustTestAccClient()
		betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
		if err != nil {
			return fmt.Errorf("failed to create beta client: %v", err)
		}

		_, _, err = getSdkKey(betaClient, projectKey, environmentKey, sdkKeyKey)
		if err != nil {
			return fmt.Errorf("received an error getting SDK key: %s", err)
		}
		return nil
	}
}

func testAccCheckSdkKeyDestroy(s *terraform.State) error {
	client := mustTestAccClient()
	betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		return fmt.Errorf("failed to create beta client: %v", err)
	}

	for _, rs := range s.RootModule().Resources {
		// Shared destroy checkers must skip data source addresses.
		if strings.HasPrefix(rs.Type, "data.") || rs.Type != "launchdarkly_sdk_key" {
			continue
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		environmentKey := rs.Primary.Attributes[ENVIRONMENT_KEY]
		sdkKeyKey := rs.Primary.Attributes[KEY]

		_, res, err := getSdkKey(betaClient, projectKey, environmentKey, sdkKeyKey)
		if isStatusNotFound(res) {
			continue
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("SDK key %q still exists", sdkKeyKey)
	}
	return nil
}
