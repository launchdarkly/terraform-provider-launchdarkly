package launchdarkly

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

const (
	testAccFeatureFlagWithViewKeysCreate = `
resource "launchdarkly_project" "test" {
	name = "%s"
	key  = "%s"
}

resource "launchdarkly_view" "view1" {
	project_key = launchdarkly_project.test.key
	key         = "test-view-1"
	name        = "Test View 1"
	maintainer_id = "%s"
}

resource "launchdarkly_view" "view2" {
	project_key = launchdarkly_project.test.key
	key         = "test-view-2"
	name        = "Test View 2"
	maintainer_id = "%s"
}

resource "launchdarkly_feature_flag" "test" {
	project_key = launchdarkly_project.test.key
	key         = "test-flag-with-views"
	name        = "Test Flag with Views"
	variation_type = "boolean"
	
	view_keys = [
		launchdarkly_view.view1.key,
		launchdarkly_view.view2.key
	]
	
	tags = ["test"]
}
`

	testAccFeatureFlagWithViewKeysUpdate = `
resource "launchdarkly_project" "test" {
	name = "%s"
	key  = "%s"
}

resource "launchdarkly_view" "view1" {
	project_key = launchdarkly_project.test.key
	key         = "test-view-1"
	name        = "Test View 1"
	maintainer_id = "%s"
}

resource "launchdarkly_view" "view2" {
	project_key = launchdarkly_project.test.key
	key         = "test-view-2"
	name        = "Test View 2"
	maintainer_id = "%s"
}

resource "launchdarkly_view" "view3" {
	project_key = launchdarkly_project.test.key
	key         = "test-view-3"
	name        = "Test View 3"
	maintainer_id = "%s"
}

resource "launchdarkly_feature_flag" "test" {
	project_key = launchdarkly_project.test.key
	key         = "test-flag-with-views"
	name        = "Test Flag with Views"
	variation_type = "boolean"
	
	view_keys = [
		launchdarkly_view.view1.key,
		launchdarkly_view.view3.key
	]
	
	tags = ["test"]
}
`

	testAccFeatureFlagWithViewKeysRemoved = `
resource "launchdarkly_project" "test" {
	name = "%s"
	key  = "%s"
}

resource "launchdarkly_view" "view1" {
	project_key = launchdarkly_project.test.key
	key         = "test-view-1"
	name        = "Test View 1"
	maintainer_id = "%s"
}

resource "launchdarkly_view" "view2" {
	project_key = launchdarkly_project.test.key
	key         = "test-view-2"
	name        = "Test View 2"
	maintainer_id = "%s"
}

resource "launchdarkly_view" "view3" {
	project_key = launchdarkly_project.test.key
	key         = "test-view-3"
	name        = "Test View 3"
	maintainer_id = "%s"
}

resource "launchdarkly_feature_flag" "test" {
	project_key = launchdarkly_project.test.key
	key         = "test-flag-with-views"
	name        = "Test Flag with Views"
	variation_type = "boolean"
	
	view_keys = []
	
	tags = ["test"]
}
`

	// Step 1: Create project and view first
	testAccFeatureFlagWithViewKeysNonexistentViewStep1 = `
resource "launchdarkly_project" "test" {
	name = "%s"
	key  = "%s"
}

resource "launchdarkly_view" "view1" {
	project_key = launchdarkly_project.test.key
	key         = "test-view-1"
	name        = "Test View 1"
	maintainer_id = "%s"
}
`

	// Step 2: Add flag with nonexistent view (should fail)
	testAccFeatureFlagWithViewKeysNonexistentViewStep2 = `
resource "launchdarkly_project" "test" {
	name = "%s"
	key  = "%s"
}

resource "launchdarkly_view" "view1" {
	project_key = launchdarkly_project.test.key
	key         = "test-view-1"
	name        = "Test View 1"
	maintainer_id = "%s"
}

resource "launchdarkly_feature_flag" "test" {
	project_key = launchdarkly_project.test.key
	key         = "test-flag-bad-view"
	name        = "Test Flag with Bad View"
	variation_type = "boolean"
	
	view_keys = [
		launchdarkly_view.view1.key,
		"nonexistent-view"
	]
}
`
)

func TestAccFeatureFlagViewKeys_CreateAndUpdate(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectName := "flag-view-keys-test-" + projectKey
	resourceName := "launchdarkly_feature_flag.test"

	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)

	members, _, err := client.ld.AccountMembersApi.GetMembers(client.ctx).Execute()
	require.NoError(t, err)
	require.True(t, len(members.Items) > 0, "This test requires at least one member in the account")

	maintainerId := members.Items[0].Id

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckFeatureFlagDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccFeatureFlagWithViewKeysCreate, projectName, projectKey, maintainerId, maintainerId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "view_keys.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "view_keys.*", "test-view-1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "view_keys.*", "test-view-2"),
					// Verify via API that the flag is actually linked to the views
					testAccCheckFlagLinkedToViews(projectKey, "test-flag-with-views", []string{"test-view-1", "test-view-2"}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccFeatureFlagWithViewKeysUpdate, projectName, projectKey, maintainerId, maintainerId, maintainerId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "view_keys.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "view_keys.*", "test-view-1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "view_keys.*", "test-view-3"),
					// Verify via API that view-2 was unlinked and view-3 was linked
					testAccCheckFlagLinkedToViews(projectKey, "test-flag-with-views", []string{"test-view-1", "test-view-3"}),
				),
			},
			{
				Config: fmt.Sprintf(testAccFeatureFlagWithViewKeysRemoved, projectName, projectKey, maintainerId, maintainerId, maintainerId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "view_keys.#", "0"),
					// Verify via API that all views were unlinked
					testAccCheckFlagLinkedToViews(projectKey, "test-flag-with-views", []string{}),
				),
			},
		},
	})
}

func TestAccFeatureFlagViewKeys_NonexistentView(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectName := "flag-bad-view-test-" + projectKey

	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)

	members, _, err := client.ld.AccountMembersApi.GetMembers(client.ctx).Execute()
	require.NoError(t, err)
	require.True(t, len(members.Items) > 0, "This test requires at least one member in the account")

	maintainerId := members.Items[0].Id

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckFeatureFlagDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create project and view first
			{
				Config: fmt.Sprintf(testAccFeatureFlagWithViewKeysNonexistentViewStep1, projectName, projectKey, maintainerId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
				),
			},
			// Step 2: Try to create flag with nonexistent view (should fail)
			{
				PreConfig: func() {
					// Wait for view to be fully propagated before attempting flag creation
					time.Sleep(3 * time.Second)
				},
				Config:      fmt.Sprintf(testAccFeatureFlagWithViewKeysNonexistentViewStep2, projectName, projectKey, maintainerId),
				ExpectError: regexp.MustCompile("view does not exist"),
			},
		},
	})
}

// testAccCheckFlagLinkedToViews verifies that a flag is linked to specific views via API
func testAccCheckFlagLinkedToViews(projectKey, flagKey string, expectedViewKeys []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*Client)
		betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
		if err != nil {
			return fmt.Errorf("failed to create beta client: %v", err)
		}

		// Get views containing this flag
		viewKeys, err := getViewsContainingFlag(betaClient, projectKey, flagKey)
		if err != nil {
			return fmt.Errorf("failed to get views for flag: %v", err)
		}

		// Verify the exact set of views matches expectations
		if len(viewKeys) != len(expectedViewKeys) {
			return fmt.Errorf("expected flag to be linked to %d views, but got %d. Expected: %v, Actual: %v",
				len(expectedViewKeys), len(viewKeys), expectedViewKeys, viewKeys)
		}

		// Check that all expected views are present
		expectedSet := make(map[string]bool)
		for _, view := range expectedViewKeys {
			expectedSet[view] = true
		}

		for _, view := range viewKeys {
			if !expectedSet[view] {
				return fmt.Errorf("unexpected view %q found for flag. Expected: %v, Actual: %v",
					view, expectedViewKeys, viewKeys)
			}
		}

		return nil
	}
}

// testAccCheckFeatureFlagDestroy verifies the flag has been destroyed
func testAccCheckFeatureFlagDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_feature_flag" {
			continue
		}

		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		flagKey := rs.Primary.Attributes[KEY]

		var res *http.Response
		var err error
		err = client.withConcurrency(client.ctx, func() error {
			_, res, err = client.ld.FeatureFlagsApi.GetFeatureFlag(client.ctx, projectKey, flagKey).Execute()
			return err
		})

		if isStatusNotFound(res) {
			continue
		}

		if err != nil {
			return err
		}

		return fmt.Errorf("feature flag %s still exists", flagKey)
	}
	return nil
}
