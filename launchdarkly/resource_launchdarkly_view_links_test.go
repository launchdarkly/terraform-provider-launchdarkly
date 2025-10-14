package launchdarkly

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

const (
	testAccViewLinksCreate = `
resource "launchdarkly_project" "test" {
	name = "%s"
	key  = "%s"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_view" "test" {
	project_key = launchdarkly_project.test.key
	key         = "test-view"
	name        = "Test View"
	description = "Test view for link testing"
	maintainer_id = "%s"
}

resource "launchdarkly_feature_flag" "test1" {
	project_key = launchdarkly_project.test.key
	key         = "test-flag-1"
	name        = "Test Flag 1"
	variation_type = "boolean"
}

resource "launchdarkly_feature_flag" "test2" {
	project_key = launchdarkly_project.test.key
	key         = "test-flag-2"
	name        = "Test Flag 2"
	variation_type = "boolean"
}

resource "launchdarkly_view_links" "test" {
	project_key = launchdarkly_project.test.key
	view_key    = launchdarkly_view.test.key
	
	flags = [
		launchdarkly_feature_flag.test1.key,
		launchdarkly_feature_flag.test2.key
	]
}
`

	testAccViewLinksUpdate = `
resource "launchdarkly_project" "test" {
	name = "%s"
	key  = "%s"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_view" "test" {
	project_key = launchdarkly_project.test.key
	key         = "test-view"
	name        = "Test View"
	description = "Test view for link testing"
	maintainer_id = "%s"
}

resource "launchdarkly_feature_flag" "test1" {
	project_key = launchdarkly_project.test.key
	key         = "test-flag-1"
	name        = "Test Flag 1"
	variation_type = "boolean"
}

resource "launchdarkly_feature_flag" "test2" {
	project_key = launchdarkly_project.test.key
	key         = "test-flag-2"
	name        = "Test Flag 2"
	variation_type = "boolean"
}

resource "launchdarkly_feature_flag" "test3" {
	project_key = launchdarkly_project.test.key
	key         = "test-flag-3"
	name        = "Test Flag 3"
	variation_type = "boolean"
}

resource "launchdarkly_view_links" "test" {
	project_key = launchdarkly_project.test.key
	view_key    = launchdarkly_view.test.key
	
	flags = [
		launchdarkly_feature_flag.test1.key,
		launchdarkly_feature_flag.test3.key
	]
}
`

	testAccViewLinksDelete = `
resource "launchdarkly_project" "test" {
	name = "%s"
	key  = "%s"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_view" "test" {
	project_key = launchdarkly_project.test.key
	key         = "test-view"
	name        = "Test View"
	description = "Test view for link testing"
	maintainer_id = "%s"
}

resource "launchdarkly_feature_flag" "test1" {
	project_key = launchdarkly_project.test.key
	key         = "test-flag-1"
	name        = "Test Flag 1"
	variation_type = "boolean"
}

resource "launchdarkly_feature_flag" "test2" {
	project_key = launchdarkly_project.test.key
	key         = "test-flag-2"
	name        = "Test Flag 2"
	variation_type = "boolean"
}

resource "launchdarkly_feature_flag" "test3" {
	project_key = launchdarkly_project.test.key
	key         = "test-flag-3"
	name        = "Test Flag 3"
	variation_type = "boolean"
}

// Note: No launchdarkly_view_links resource - this should unlink all flags
`

	testAccViewLinksWithSegmentsCreate = `
resource "launchdarkly_project" "test" {
	name = "%s"
	key  = "%s"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
	environments {
		name  = "Production"
		key   = "production"
		color = "AAAAAA"
	}
}

resource "launchdarkly_view" "test" {
	project_key   = launchdarkly_project.test.key
	key           = "test-view"
	name          = "Test View"
	description   = "Test view for flags and segments"
	maintainer_id = "%s"
}

resource "launchdarkly_feature_flag" "flag1" {
	project_key    = launchdarkly_project.test.key
	key            = "test-flag-1"
	name           = "Test Flag 1"
	variation_type = "boolean"
}

resource "launchdarkly_feature_flag" "flag2" {
	project_key    = launchdarkly_project.test.key
	key            = "test-flag-2"
	name           = "Test Flag 2"
	variation_type = "boolean"
}

resource "launchdarkly_feature_flag" "flag3" {
	project_key    = launchdarkly_project.test.key
	key            = "test-flag-3"
	name           = "Test Flag 3"
	variation_type = "boolean"
}

resource "launchdarkly_segment" "segment1" {
	key         = "test-segment-1"
	project_key = launchdarkly_project.test.key
	env_key     = "test-env"
	name        = "Test Segment 1"
	description = "Test segment 1"
}

resource "launchdarkly_segment" "segment2" {
	key         = "test-segment-2"
	project_key = launchdarkly_project.test.key
	env_key     = "test-env"
	name        = "Test Segment 2"
	description = "Test segment 2"
}

resource "launchdarkly_segment" "segment3" {
	key         = "test-segment-3"
	project_key = launchdarkly_project.test.key
	env_key     = "production"
	name        = "Test Segment 3"
	description = "Test segment 3"
}

resource "launchdarkly_view_links" "test" {
	project_key = launchdarkly_project.test.key
	view_key    = launchdarkly_view.test.key
	
	flags = [
		launchdarkly_feature_flag.flag1.key,
		launchdarkly_feature_flag.flag2.key
	]

	segments {
		environment_id = launchdarkly_project.test.environments[0].client_side_id
		segment_key    = launchdarkly_segment.segment1.key
	}

	segments {
		environment_id = launchdarkly_project.test.environments[0].client_side_id
		segment_key    = launchdarkly_segment.segment2.key
	}
}
`

	testAccViewLinksWithSegmentsUpdate = `
resource "launchdarkly_project" "test" {
	name = "%s"
	key  = "%s"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
	environments {
		name  = "Production"
		key   = "production"
		color = "AAAAAA"
	}
}

resource "launchdarkly_view" "test" {
	project_key   = launchdarkly_project.test.key
	key           = "test-view"
	name          = "Test View"
	description   = "Test view for flags and segments"
	maintainer_id = "%s"
}

resource "launchdarkly_feature_flag" "flag1" {
	project_key    = launchdarkly_project.test.key
	key            = "test-flag-1"
	name           = "Test Flag 1"
	variation_type = "boolean"
}

resource "launchdarkly_feature_flag" "flag2" {
	project_key    = launchdarkly_project.test.key
	key            = "test-flag-2"
	name           = "Test Flag 2"
	variation_type = "boolean"
}

resource "launchdarkly_feature_flag" "flag3" {
	project_key    = launchdarkly_project.test.key
	key            = "test-flag-3"
	name           = "Test Flag 3"
	variation_type = "boolean"
}

resource "launchdarkly_segment" "segment1" {
	key         = "test-segment-1"
	project_key = launchdarkly_project.test.key
	env_key     = "test-env"
	name        = "Test Segment 1"
	description = "Test segment 1"
}

resource "launchdarkly_segment" "segment2" {
	key         = "test-segment-2"
	project_key = launchdarkly_project.test.key
	env_key     = "test-env"
	name        = "Test Segment 2"
	description = "Test segment 2"
}

resource "launchdarkly_segment" "segment3" {
	key         = "test-segment-3"
	project_key = launchdarkly_project.test.key
	env_key     = "production"
	name        = "Test Segment 3"
	description = "Test segment 3"
}

resource "launchdarkly_view_links" "test" {
	project_key = launchdarkly_project.test.key
	view_key    = launchdarkly_view.test.key
	
	flags = [
		launchdarkly_feature_flag.flag1.key,
		launchdarkly_feature_flag.flag3.key
	]

	segments {
		environment_id = launchdarkly_project.test.environments[0].client_side_id
		segment_key    = launchdarkly_segment.segment1.key
	}

	segments {
		environment_id = launchdarkly_project.test.environments[1].client_side_id
		segment_key    = launchdarkly_segment.segment3.key
	}
}
`

	testAccViewLinksWithSegmentsFinal = `
resource "launchdarkly_project" "test" {
	name = "%s"
	key  = "%s"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
	environments {
		name  = "Production"
		key   = "production"
		color = "AAAAAA"
	}
}

resource "launchdarkly_view" "test" {
	project_key   = launchdarkly_project.test.key
	key           = "test-view"
	name          = "Test View"
	description   = "Test view for flags and segments"
	maintainer_id = "%s"
}

resource "launchdarkly_feature_flag" "flag1" {
	project_key    = launchdarkly_project.test.key
	key            = "test-flag-1"
	name           = "Test Flag 1"
	variation_type = "boolean"
}

resource "launchdarkly_feature_flag" "flag2" {
	project_key    = launchdarkly_project.test.key
	key            = "test-flag-2"
	name           = "Test Flag 2"
	variation_type = "boolean"
}

resource "launchdarkly_feature_flag" "flag3" {
	project_key    = launchdarkly_project.test.key
	key            = "test-flag-3"
	name           = "Test Flag 3"
	variation_type = "boolean"
}

resource "launchdarkly_segment" "segment1" {
	key         = "test-segment-1"
	project_key = launchdarkly_project.test.key
	env_key     = "test-env"
	name        = "Test Segment 1"
	description = "Test segment 1"
}

resource "launchdarkly_segment" "segment2" {
	key         = "test-segment-2"
	project_key = launchdarkly_project.test.key
	env_key     = "test-env"
	name        = "Test Segment 2"
	description = "Test segment 2"
}

resource "launchdarkly_segment" "segment3" {
	key         = "test-segment-3"
	project_key = launchdarkly_project.test.key
	env_key     = "production"
	name        = "Test Segment 3"
	description = "Test segment 3"
}

resource "launchdarkly_view_links" "test" {
	project_key = launchdarkly_project.test.key
	view_key    = launchdarkly_view.test.key
	
	flags = [
		launchdarkly_feature_flag.flag2.key,
		launchdarkly_feature_flag.flag3.key
	]

	segments {
		environment_id = launchdarkly_project.test.environments[0].client_side_id
		segment_key    = launchdarkly_segment.segment2.key
	}

	segments {
		environment_id = launchdarkly_project.test.environments[1].client_side_id
		segment_key    = launchdarkly_segment.segment3.key
	}

	segments {
		environment_id = launchdarkly_project.test.environments[0].client_side_id
		segment_key    = launchdarkly_segment.segment1.key
	}
}
`
)

func TestAccViewLinks_Update(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectName := "view-links-test-" + projectKey
	resourceName := "launchdarkly_view_links.test"

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
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccViewLinksCreate, projectName, projectKey, maintainerId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewForLinksExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "flags.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "flags.*", "test-flag-1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "flags.*", "test-flag-2"),
					testAccCheckViewLinksAPIState(projectKey, "test-view", []string{"test-flag-1", "test-flag-2"}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccViewLinksUpdate, projectName, projectKey, maintainerId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewForLinksExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "flags.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "flags.*", "test-flag-1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "flags.*", "test-flag-3"),
					testAccCheckViewLinksAPIState(projectKey, "test-view", []string{"test-flag-1", "test-flag-3"}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccViewLinksDelete, projectName, projectKey, maintainerId),
				Check: resource.ComposeTestCheckFunc(
					// Verify that the view still exists but has no linked flags
					testAccCheckViewExistsViaAPI(projectKey, "test-view"),
					testAccCheckViewLinksAPIState(projectKey, "test-view", []string{}),
				),
			},
		},
	})
}

func TestAccViewLinks_Import(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectName := "view-links-test-" + projectKey
	resourceName := "launchdarkly_view_links.test"

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
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccViewLinksCreate, projectName, projectKey, maintainerId),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccViewLinks_WithFlagsAndSegments(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectName := "view-links-mixed-test-" + projectKey
	resourceName := "launchdarkly_view_links.test"

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
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccViewLinksWithSegmentsCreate, projectName, projectKey, maintainerId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewForLinksExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "flags.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "flags.*", "test-flag-1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "flags.*", "test-flag-2"),
					resource.TestCheckResourceAttr(resourceName, "segments.#", "2"),
					testAccCheckViewLinksAPIState(projectKey, "test-view", []string{"test-flag-1", "test-flag-2"}),
					testAccCheckViewLinksSegmentsAPIState(projectKey, "test-view", []string{"test-segment-1", "test-segment-2"}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccViewLinksWithSegmentsUpdate, projectName, projectKey, maintainerId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewForLinksExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "flags.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "flags.*", "test-flag-1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "flags.*", "test-flag-3"),
					resource.TestCheckResourceAttr(resourceName, "segments.#", "2"),
					testAccCheckViewLinksAPIState(projectKey, "test-view", []string{"test-flag-1", "test-flag-3"}),
					testAccCheckViewLinksSegmentsAPIState(projectKey, "test-view", []string{"test-segment-1", "test-segment-3"}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccViewLinksWithSegmentsFinal, projectName, projectKey, maintainerId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewForLinksExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "flags.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "flags.*", "test-flag-2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "flags.*", "test-flag-3"),
					resource.TestCheckResourceAttr(resourceName, "segments.#", "3"),
					testAccCheckViewLinksAPIState(projectKey, "test-view", []string{"test-flag-2", "test-flag-3"}),
					testAccCheckViewLinksSegmentsAPIState(projectKey, "test-view", []string{"test-segment-1", "test-segment-2", "test-segment-3"}),
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

func testAccCheckViewForLinksExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("view links ID is not set")
		}

		client := testAccProvider.Meta().(*Client)
		betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
		if err != nil {
			return err
		}

		projectKey := rs.Primary.Attributes["project_key"]
		viewKey := rs.Primary.Attributes["view_key"]

		exists, err := viewExists(projectKey, viewKey, betaClient)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("view %s/%s does not exist", projectKey, viewKey)
		}

		return nil
	}
}

// testAccCheckViewExistsViaAPI verifies that a view exists via direct API call
func testAccCheckViewExistsViaAPI(projectKey, viewKey string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*Client)
		betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
		if err != nil {
			return fmt.Errorf("failed to create beta client: %v", err)
		}

		exists, err := viewExists(projectKey, viewKey, betaClient)
		if err != nil {
			return fmt.Errorf("error checking if view exists: %v", err)
		}
		if !exists {
			return fmt.Errorf("view %s/%s does not exist", projectKey, viewKey)
		}

		return nil
	}
}

// testAccCheckViewLinksAPIState verifies the actual linked flags via API call
func testAccCheckViewLinksAPIState(projectKey, viewKey string, expectedFlags []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*Client)
		betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
		if err != nil {
			return fmt.Errorf("failed to create beta client: %v", err)
		}

		// Get linked flags from API
		linkedResources, err := getLinkedResources(betaClient, projectKey, viewKey, FLAGS)
		if err != nil {
			return fmt.Errorf("failed to get linked resources: %v", err)
		}

		// Extract flag keys from the response
		actualFlags := make([]string, len(linkedResources))
		for i, resource := range linkedResources {
			actualFlags[i] = resource.ResourceKey
		}

		// Verify the exact set of flags matches expectations
		if len(actualFlags) != len(expectedFlags) {
			return fmt.Errorf("expected %d linked flags, but got %d. Expected: %v, Actual: %v",
				len(expectedFlags), len(actualFlags), expectedFlags, actualFlags)
		}

		// Check that all expected flags are present
		expectedSet := make(map[string]bool)
		for _, flag := range expectedFlags {
			expectedSet[flag] = true
		}

		for _, flag := range actualFlags {
			if !expectedSet[flag] {
				return fmt.Errorf("unexpected flag %q found in linked flags. Expected: %v, Actual: %v",
					flag, expectedFlags, actualFlags)
			}
		}

		// Check that no extra flags are present (this is redundant with length check, but explicit)
		actualSet := make(map[string]bool)
		for _, flag := range actualFlags {
			actualSet[flag] = true
		}

		for _, flag := range expectedFlags {
			if !actualSet[flag] {
				return fmt.Errorf("expected flag %q not found in linked flags. Expected: %v, Actual: %v",
					flag, expectedFlags, actualFlags)
			}
		}

		return nil
	}
}

// testAccCheckViewLinksSegmentsAPIState verifies the actual linked segments via API call
func testAccCheckViewLinksSegmentsAPIState(projectKey, viewKey string, expectedSegments []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*Client)
		betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
		if err != nil {
			return fmt.Errorf("failed to create beta client: %v", err)
		}

		// Get linked segments from API
		linkedResources, err := getLinkedResources(betaClient, projectKey, viewKey, SEGMENTS)
		if err != nil {
			return fmt.Errorf("failed to get linked segments: %v", err)
		}

		// Extract segment keys from the response
		actualSegments := make([]string, len(linkedResources))
		for i, resource := range linkedResources {
			actualSegments[i] = resource.ResourceKey
		}

		// Verify the exact set of segments matches expectations
		if len(actualSegments) != len(expectedSegments) {
			return fmt.Errorf("expected %d linked segments, but got %d. Expected: %v, Actual: %v",
				len(expectedSegments), len(actualSegments), expectedSegments, actualSegments)
		}

		// Check that all expected segments are present
		expectedSet := make(map[string]bool)
		for _, segment := range expectedSegments {
			expectedSet[segment] = true
		}

		for _, segment := range actualSegments {
			if !expectedSet[segment] {
				return fmt.Errorf("unexpected segment %q found in linked segments. Expected: %v, Actual: %v",
					segment, expectedSegments, actualSegments)
			}
		}

		// Check that no expected segments are missing
		actualSet := make(map[string]bool)
		for _, segment := range actualSegments {
			actualSet[segment] = true
		}

		for _, segment := range expectedSegments {
			if !actualSet[segment] {
				return fmt.Errorf("expected segment %q not found in linked segments. Expected: %v, Actual: %v",
					segment, expectedSegments, actualSegments)
			}
		}

		return nil
	}
}
