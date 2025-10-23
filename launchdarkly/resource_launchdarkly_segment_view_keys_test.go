package launchdarkly

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	ldapi "github.com/launchdarkly/api-client-go/v17"
	"github.com/stretchr/testify/require"
)

const (
	testAccSegmentWithViewKeysCreate = `
resource "launchdarkly_project" "test" {
	name = "%s"
	key  = "%s"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
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

resource "launchdarkly_segment" "test" {
	project_key = launchdarkly_project.test.key
	env_key     = "test-env"
	key         = "test-segment-with-views"
	name        = "Test Segment with Views"
	description = "Test segment"
	
	view_keys = [
		launchdarkly_view.view1.key,
		launchdarkly_view.view2.key
	]
	
	tags = ["test"]
}
`

	testAccSegmentWithViewKeysUpdate = `
resource "launchdarkly_project" "test" {
	name = "%s"
	key  = "%s"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
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

resource "launchdarkly_segment" "test" {
	project_key = launchdarkly_project.test.key
	env_key     = "test-env"
	key         = "test-segment-with-views"
	name        = "Test Segment with Views"
	description = "Test segment"
	
	view_keys = [
		launchdarkly_view.view1.key,
		launchdarkly_view.view3.key
	]
	
	tags = ["test"]
}
`

	testAccSegmentWithViewKeysRemoved = `
resource "launchdarkly_project" "test" {
	name = "%s"
	key  = "%s"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
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

resource "launchdarkly_segment" "test" {
	project_key = launchdarkly_project.test.key
	env_key     = "test-env"
	key         = "test-segment-with-views"
	name        = "Test Segment with Views"
	description = "Test segment"
	
	tags = ["test"]
}
`

	testAccSegmentWithViewKeysNonexistentView = `
resource "launchdarkly_project" "test" {
	name = "%s"
	key  = "%s"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_view" "view1" {
	project_key = launchdarkly_project.test.key
	key         = "test-view-1"
	name        = "Test View 1"
	maintainer_id = "%s"
}

resource "launchdarkly_segment" "test" {
	project_key = launchdarkly_project.test.key
	env_key     = "test-env"
	key         = "test-segment-bad-view"
	name        = "Test Segment with Bad View"
	description = "Test segment"
	
	view_keys = [
		launchdarkly_view.view1.key,
		"nonexistent-view"
	]
}
`
)

func TestAccSegmentViewKeys_CreateAndUpdate(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectName := "segment-view-keys-test-" + projectKey
	resourceName := "launchdarkly_segment.test"

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
		CheckDestroy: testAccCheckSegmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccSegmentWithViewKeysCreate, projectName, projectKey, maintainerId, maintainerId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "view_keys.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "view_keys.*", "test-view-1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "view_keys.*", "test-view-2"),
					// Verify via API that the segment is actually linked to the views
					testAccCheckSegmentLinkedToViews(projectKey, "test-env", "test-segment-with-views", []string{"test-view-1", "test-view-2"}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccSegmentWithViewKeysUpdate, projectName, projectKey, maintainerId, maintainerId, maintainerId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "view_keys.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "view_keys.*", "test-view-1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "view_keys.*", "test-view-3"),
					// Verify via API that view-2 was unlinked and view-3 was linked
					testAccCheckSegmentLinkedToViews(projectKey, "test-env", "test-segment-with-views", []string{"test-view-1", "test-view-3"}),
				),
			},
			{
				Config: fmt.Sprintf(testAccSegmentWithViewKeysRemoved, projectName, projectKey, maintainerId, maintainerId, maintainerId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "view_keys.#", "0"),
					// Verify via API that all views were unlinked
					testAccCheckSegmentLinkedToViews(projectKey, "test-env", "test-segment-with-views", []string{}),
				),
			},
		},
	})
}

func TestAccSegmentViewKeys_NonexistentView(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectName := "segment-bad-view-test-" + projectKey

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
		CheckDestroy: testAccCheckSegmentDestroy,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccSegmentWithViewKeysNonexistentView, projectName, projectKey, maintainerId),
				ExpectError: regexp.MustCompile("view does not exist"),
			},
		},
	})
}

// testAccCheckSegmentLinkedToViews verifies that a segment is linked to specific views via API
func testAccCheckSegmentLinkedToViews(projectKey, envKey, segmentKey string, expectedViewKeys []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*Client)
		betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
		if err != nil {
			return fmt.Errorf("failed to create beta client: %v", err)
		}

		// Get the environment to retrieve its ID
		var env *ldapi.Environment
		err = client.withConcurrency(client.ctx, func() error {
			env, _, err = client.ld.EnvironmentsApi.GetEnvironment(client.ctx, projectKey, envKey).Execute()
			return err
		})
		if err != nil {
			return fmt.Errorf("failed to get environment: %v", err)
		}

		// Get views containing this segment
		viewKeys, err := getViewsContainingSegment(betaClient, projectKey, env.Id, segmentKey)
		if err != nil {
			return fmt.Errorf("failed to get views for segment: %v", err)
		}

		// Verify the exact set of views matches expectations
		if len(viewKeys) != len(expectedViewKeys) {
			return fmt.Errorf("expected segment to be linked to %d views, but got %d. Expected: %v, Actual: %v",
				len(expectedViewKeys), len(viewKeys), expectedViewKeys, viewKeys)
		}

		// Check that all expected views are present
		expectedSet := make(map[string]bool)
		for _, view := range expectedViewKeys {
			expectedSet[view] = true
		}

		for _, view := range viewKeys {
			if !expectedSet[view] {
				return fmt.Errorf("unexpected view %q found for segment. Expected: %v, Actual: %v",
					view, expectedViewKeys, viewKeys)
			}
		}

		return nil
	}
}

// testAccCheckSegmentDestroy verifies the segment has been destroyed
func testAccCheckSegmentDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_segment" {
			continue
		}

		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		envKey := rs.Primary.Attributes[ENV_KEY]
		segmentKey := rs.Primary.Attributes[KEY]

		var res *http.Response
		var err error
		err = client.withConcurrency(client.ctx, func() error {
			_, res, err = client.ld.SegmentsApi.GetSegment(client.ctx, projectKey, envKey, segmentKey).Execute()
			return err
		})

		if isStatusNotFound(res) {
			continue
		}

		if err != nil {
			return err
		}

		return fmt.Errorf("segment %s still exists", segmentKey)
	}
	return nil
}
