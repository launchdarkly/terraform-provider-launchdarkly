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
	testAccViewFilterLinksCreateFlagFilter = `
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
	project_key   = launchdarkly_project.test.key
	key           = "test-view"
	name          = "Test View"
	description   = "Test view for filter link testing"
	maintainer_id = "%s"
}

resource "launchdarkly_feature_flag" "test1" {
	project_key    = launchdarkly_project.test.key
	key            = "filter-test-flag-1"
	name           = "Filter Test Flag 1"
	variation_type = "boolean"
	tags           = ["filter-test"]
}

resource "launchdarkly_feature_flag" "test2" {
	project_key    = launchdarkly_project.test.key
	key            = "filter-test-flag-2"
	name           = "Filter Test Flag 2"
	variation_type = "boolean"
	tags           = ["filter-test"]
}

resource "launchdarkly_view_filter_links" "test" {
	project_key = launchdarkly_project.test.key
	view_key    = launchdarkly_view.test.key
	flag_filter = "tags:filter-test"

	depends_on = [
		launchdarkly_feature_flag.test1,
		launchdarkly_feature_flag.test2
	]
}
`

	testAccViewFilterLinksUpdateFlagFilter = `
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
	project_key   = launchdarkly_project.test.key
	key           = "test-view"
	name          = "Test View"
	description   = "Test view for filter link testing"
	maintainer_id = "%s"
}

resource "launchdarkly_feature_flag" "test1" {
	project_key    = launchdarkly_project.test.key
	key            = "filter-test-flag-1"
	name           = "Filter Test Flag 1"
	variation_type = "boolean"
	tags           = ["filter-test"]
}

resource "launchdarkly_feature_flag" "test2" {
	project_key    = launchdarkly_project.test.key
	key            = "filter-test-flag-2"
	name           = "Filter Test Flag 2"
	variation_type = "boolean"
	tags           = ["filter-test"]
}

resource "launchdarkly_feature_flag" "test3" {
	project_key    = launchdarkly_project.test.key
	key            = "filter-test-flag-3"
	name           = "Filter Test Flag 3"
	variation_type = "boolean"
	tags           = ["filter-test-v2"]
}

resource "launchdarkly_view_filter_links" "test" {
	project_key = launchdarkly_project.test.key
	view_key    = launchdarkly_view.test.key
	flag_filter = "tags:filter-test-v2"

	depends_on = [
		launchdarkly_feature_flag.test1,
		launchdarkly_feature_flag.test2,
		launchdarkly_feature_flag.test3
	]
}
`

	testAccViewFilterLinksRelinkAfterTagRemoval = `
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
	project_key   = launchdarkly_project.test.key
	key           = "test-view"
	name          = "Test View"
	description   = "Test view for filter link testing"
	maintainer_id = "%s"
}

resource "launchdarkly_feature_flag" "test1" {
	project_key    = launchdarkly_project.test.key
	key            = "filter-test-flag-1"
	name           = "Filter Test Flag 1"
	variation_type = "boolean"
	tags           = ["filter-test"]
}

resource "launchdarkly_feature_flag" "test2" {
	project_key    = launchdarkly_project.test.key
	key            = "filter-test-flag-2"
	name           = "Filter Test Flag 2"
	variation_type = "boolean"
	tags           = ["other-tag"]
}

resource "launchdarkly_feature_flag" "test3" {
	project_key    = launchdarkly_project.test.key
	key            = "filter-test-flag-3"
	name           = "Filter Test Flag 3"
	variation_type = "boolean"
	tags           = ["filter-test-v2"]
}

resource "launchdarkly_view_filter_links" "test" {
	project_key = launchdarkly_project.test.key
	view_key    = launchdarkly_view.test.key
	flag_filter = "tags:filter-test"

	depends_on = [
		launchdarkly_feature_flag.test1,
		launchdarkly_feature_flag.test2,
		launchdarkly_feature_flag.test3
	]
}
`

	testAccViewFilterLinksCreateSegmentFilter = `
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
	project_key   = launchdarkly_project.test.key
	key           = "test-view"
	name          = "Test View"
	description   = "Test view for segment filter link testing"
	maintainer_id = "%s"
}

resource "launchdarkly_segment" "test1" {
	key         = "filter-test-segment-1"
	project_key = launchdarkly_project.test.key
	env_key     = "test-env"
	name        = "Filter Test Segment 1"
	tags        = ["segment-filter-test"]
}

resource "launchdarkly_view_filter_links" "test" {
	project_key                   = launchdarkly_project.test.key
	view_key                      = launchdarkly_view.test.key
	segment_filter                = "tags anyOf [\"segment-filter-test\"]"
	segment_filter_environment_id = launchdarkly_project.test.environments[0].client_side_id

	depends_on = [
		launchdarkly_segment.test1
	]
}
`

	testAccViewFilterLinksCreateBothFilters = `
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
	project_key   = launchdarkly_project.test.key
	key           = "test-view"
	name          = "Test View"
	description   = "Test view for both filter types"
	maintainer_id = "%s"
}

resource "launchdarkly_feature_flag" "test1" {
	project_key    = launchdarkly_project.test.key
	key            = "both-filter-flag-1"
	name           = "Both Filter Flag 1"
	variation_type = "boolean"
	tags           = ["both-filter-test"]
}

resource "launchdarkly_segment" "test1" {
	key         = "both-filter-segment-1"
	project_key = launchdarkly_project.test.key
	env_key     = "test-env"
	name        = "Both Filter Segment 1"
	tags        = ["both-filter-test"]
}

resource "launchdarkly_view_filter_links" "test" {
	project_key                   = launchdarkly_project.test.key
	view_key                      = launchdarkly_view.test.key
	flag_filter                   = "tags:both-filter-test"
	segment_filter                = "tags anyOf [\"both-filter-test\"]"
	segment_filter_environment_id = launchdarkly_project.test.environments[0].client_side_id

	depends_on = [
		launchdarkly_feature_flag.test1,
		launchdarkly_segment.test1
	]
}
`

	testAccViewFilterLinksTriggersStep1 = `
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
	project_key   = launchdarkly_project.test.key
	key           = "test-view"
	name          = "Test View"
	description   = "Test view for trigger testing"
	maintainer_id = "%s"
}

resource "launchdarkly_feature_flag" "trigger1" {
	project_key    = launchdarkly_project.test.key
	key            = "trigger-flag-1"
	name           = "Trigger Flag 1"
	variation_type = "boolean"
	tags           = ["trigger-test"]
}

resource "launchdarkly_feature_flag" "trigger2" {
	project_key    = launchdarkly_project.test.key
	key            = "trigger-flag-2"
	name           = "Trigger Flag 2"
	variation_type = "boolean"
	tags           = ["trigger-test"]
}

resource "launchdarkly_view_filter_links" "test" {
	project_key = launchdarkly_project.test.key
	view_key    = launchdarkly_view.test.key
	flag_filter = "tags:trigger-test"
	reconcile_on_apply = true

	depends_on = [
		launchdarkly_feature_flag.trigger1,
		launchdarkly_feature_flag.trigger2
	]
}
`

	testAccViewFilterLinksTriggersStep2 = `
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
	project_key   = launchdarkly_project.test.key
	key           = "test-view"
	name          = "Test View"
	description   = "Test view for trigger testing"
	maintainer_id = "%s"
}

resource "launchdarkly_feature_flag" "trigger1" {
	project_key    = launchdarkly_project.test.key
	key            = "trigger-flag-1"
	name           = "Trigger Flag 1"
	variation_type = "boolean"
	tags           = ["trigger-test"]
}

resource "launchdarkly_feature_flag" "trigger2" {
	project_key    = launchdarkly_project.test.key
	key            = "trigger-flag-2"
	name           = "Trigger Flag 2"
	variation_type = "boolean"
	tags           = ["other-tag"]
}

resource "launchdarkly_view_filter_links" "test" {
	project_key = launchdarkly_project.test.key
	view_key    = launchdarkly_view.test.key
	flag_filter = "tags:trigger-test"
	reconcile_on_apply = true

	depends_on = [
		launchdarkly_feature_flag.trigger1,
		launchdarkly_feature_flag.trigger2
	]
}
`

	testAccViewFilterLinksToggleStep1 = `
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
	project_key   = launchdarkly_project.test.key
	key           = "test-view"
	name          = "Test View"
	description   = "Test view for reconcile toggle testing"
	maintainer_id = "%s"
}

resource "launchdarkly_feature_flag" "toggle1" {
	project_key    = launchdarkly_project.test.key
	key            = "toggle-flag-1"
	name           = "Toggle Flag 1"
	variation_type = "boolean"
	tags           = ["toggle-test"]
}

resource "launchdarkly_feature_flag" "toggle2" {
	project_key    = launchdarkly_project.test.key
	key            = "toggle-flag-2"
	name           = "Toggle Flag 2"
	variation_type = "boolean"
	tags           = ["toggle-test"]
}

resource "launchdarkly_view_filter_links" "test" {
	project_key         = launchdarkly_project.test.key
	view_key            = launchdarkly_view.test.key
	flag_filter         = "tags:toggle-test"
	reconcile_on_apply  = false

	depends_on = [
		launchdarkly_feature_flag.toggle1,
		launchdarkly_feature_flag.toggle2
	]
}
`

	testAccViewFilterLinksToggleStep2 = `
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
	project_key   = launchdarkly_project.test.key
	key           = "test-view"
	name          = "Test View"
	description   = "Test view for reconcile toggle testing"
	maintainer_id = "%s"
}

resource "launchdarkly_feature_flag" "toggle1" {
	project_key    = launchdarkly_project.test.key
	key            = "toggle-flag-1"
	name           = "Toggle Flag 1"
	variation_type = "boolean"
	tags           = ["toggle-test"]
}

resource "launchdarkly_feature_flag" "toggle2" {
	project_key    = launchdarkly_project.test.key
	key            = "toggle-flag-2"
	name           = "Toggle Flag 2"
	variation_type = "boolean"
	tags           = ["toggle-test"]
}

resource "launchdarkly_view_filter_links" "test" {
	project_key         = launchdarkly_project.test.key
	view_key            = launchdarkly_view.test.key
	flag_filter         = "tags:toggle-test"
	reconcile_on_apply  = true

	depends_on = [
		launchdarkly_feature_flag.toggle1,
		launchdarkly_feature_flag.toggle2
	]
}
`

	testAccViewFilterLinksSegmentTriggersStep1 = `
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
	project_key   = launchdarkly_project.test.key
	key           = "test-view"
	name          = "Test View"
	description   = "Test view for segment trigger testing"
	maintainer_id = "%s"
}

resource "launchdarkly_segment" "trigger1" {
	key         = "segment-trigger-1"
	project_key = launchdarkly_project.test.key
	env_key     = "test-env"
	name        = "Segment Trigger 1"
	tags        = ["segment-trigger-test"]
}

resource "launchdarkly_segment" "trigger2" {
	key         = "segment-trigger-2"
	project_key = launchdarkly_project.test.key
	env_key     = "test-env"
	name        = "Segment Trigger 2"
	tags        = ["segment-trigger-test"]
}

resource "launchdarkly_view_filter_links" "test" {
	project_key                   = launchdarkly_project.test.key
	view_key                      = launchdarkly_view.test.key
	segment_filter                = "tags anyOf [\"segment-trigger-test\"]"
	segment_filter_environment_id = launchdarkly_project.test.environments[0].client_side_id
	reconcile_on_apply            = true

	depends_on = [
		launchdarkly_segment.trigger1,
		launchdarkly_segment.trigger2
	]
}
`

	testAccViewFilterLinksSegmentTriggersStep2 = `
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
	project_key   = launchdarkly_project.test.key
	key           = "test-view"
	name          = "Test View"
	description   = "Test view for segment trigger testing"
	maintainer_id = "%s"
}

resource "launchdarkly_segment" "trigger1" {
	key         = "segment-trigger-1"
	project_key = launchdarkly_project.test.key
	env_key     = "test-env"
	name        = "Segment Trigger 1"
	tags        = ["segment-trigger-test"]
}

resource "launchdarkly_segment" "trigger2" {
	key         = "segment-trigger-2"
	project_key = launchdarkly_project.test.key
	env_key     = "test-env"
	name        = "Segment Trigger 2"
	tags        = ["other-tag"]
}

resource "launchdarkly_view_filter_links" "test" {
	project_key                   = launchdarkly_project.test.key
	view_key                      = launchdarkly_view.test.key
	segment_filter                = "tags anyOf [\"segment-trigger-test\"]"
	segment_filter_environment_id = launchdarkly_project.test.environments[0].client_side_id
	reconcile_on_apply            = true

	depends_on = [
		launchdarkly_segment.trigger1,
		launchdarkly_segment.trigger2
	]
}
`

	testAccViewFilterLinksRemoveOneFilter = `
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
	project_key   = launchdarkly_project.test.key
	key           = "test-view"
	name          = "Test View"
	description   = "Test view for both filter types"
	maintainer_id = "%s"
}

resource "launchdarkly_feature_flag" "test1" {
	project_key    = launchdarkly_project.test.key
	key            = "both-filter-flag-1"
	name           = "Both Filter Flag 1"
	variation_type = "boolean"
	tags           = ["both-filter-test"]
}

resource "launchdarkly_segment" "test1" {
	key         = "both-filter-segment-1"
	project_key = launchdarkly_project.test.key
	env_key     = "test-env"
	name        = "Both Filter Segment 1"
	tags        = ["both-filter-test"]
}

resource "launchdarkly_view_filter_links" "test" {
	project_key = launchdarkly_project.test.key
	view_key    = launchdarkly_view.test.key
	flag_filter = "tags:both-filter-test"

	depends_on = [
		launchdarkly_feature_flag.test1,
		launchdarkly_segment.test1
	]
}
`
)

func TestAccViewFilterLinks_FlagFilter(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectName := "view-filter-links-test-" + projectKey
	resourceName := "launchdarkly_view_filter_links.test"

	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)

	members, _, err := client.ld.AccountMembersApi.GetMembers(client.ctx).Execute()
	require.NoError(t, err)
	require.True(t, len(members.Items) > 0, "This test requires at least one member in the account")

	maintainerID := members.Items[0].Id

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccViewFilterLinksCreateFlagFilter, projectName, projectKey, maintainerID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewForFilterLinksExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, FLAG_FILTER, "tags:filter-test"),
					resource.TestCheckNoResourceAttr(resourceName, SEGMENT_FILTER),
					// Verify flags are actually linked via the API
					testAccCheckViewLinksAPIState(projectKey, "test-view", []string{"filter-test-flag-1", "filter-test-flag-2"}),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{FLAG_FILTER, SEGMENT_FILTER, SEGMENT_FILTER_ENVIRONMENT_ID, RECONCILE_ON_APPLY, RESOLVED_AT},
			},
			{
				Config: fmt.Sprintf(testAccViewFilterLinksUpdateFlagFilter, projectName, projectKey, maintainerID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewForFilterLinksExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, FLAG_FILTER, "tags:filter-test-v2"),
					// After update: old filter flags should be unlinked, new filter flags should be linked
					testAccCheckViewLinksAPIState(projectKey, "test-view", []string{"filter-test-flag-3"}),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{FLAG_FILTER, SEGMENT_FILTER, SEGMENT_FILTER_ENVIRONMENT_ID, RECONCILE_ON_APPLY, RESOLVED_AT},
			},
			{
				// Re-apply original filter after removing the tag from flag2.
				// flag2's tag changes from "filter-test" to "other-tag", so only flag1 should match.
				Config: fmt.Sprintf(testAccViewFilterLinksRelinkAfterTagRemoval, projectName, projectKey, maintainerID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewForFilterLinksExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, FLAG_FILTER, "tags:filter-test"),
					// Only flag1 should be linked — flag2 no longer has the "filter-test" tag
					testAccCheckViewLinksAPIState(projectKey, "test-view", []string{"filter-test-flag-1"}),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{FLAG_FILTER, SEGMENT_FILTER, SEGMENT_FILTER_ENVIRONMENT_ID, RECONCILE_ON_APPLY, RESOLVED_AT},
			},
		},
	})
}

func TestAccViewFilterLinks_SegmentFilter(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectName := "view-filter-links-seg-test-" + projectKey
	resourceName := "launchdarkly_view_filter_links.test"

	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)

	members, _, err := client.ld.AccountMembersApi.GetMembers(client.ctx).Execute()
	require.NoError(t, err)
	require.True(t, len(members.Items) > 0, "This test requires at least one member in the account")

	maintainerID := members.Items[0].Id

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccViewFilterLinksCreateSegmentFilter, projectName, projectKey, maintainerID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewForFilterLinksExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, SEGMENT_FILTER, `tags anyOf ["segment-filter-test"]`),
					resource.TestCheckNoResourceAttr(resourceName, FLAG_FILTER),
					testAccCheckViewLinksSegmentsAPIState(projectKey, "test-view", []string{"filter-test-segment-1"}),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{FLAG_FILTER, SEGMENT_FILTER, SEGMENT_FILTER_ENVIRONMENT_ID, RECONCILE_ON_APPLY, RESOLVED_AT},
			},
		},
	})
}

func TestAccViewFilterLinks_BothFilters(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectName := "view-filter-links-both-test-" + projectKey
	resourceName := "launchdarkly_view_filter_links.test"

	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)

	members, _, err := client.ld.AccountMembersApi.GetMembers(client.ctx).Execute()
	require.NoError(t, err)
	require.True(t, len(members.Items) > 0, "This test requires at least one member in the account")

	maintainerID := members.Items[0].Id

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccViewFilterLinksCreateBothFilters, projectName, projectKey, maintainerID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewForFilterLinksExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, FLAG_FILTER, "tags:both-filter-test"),
					resource.TestCheckResourceAttr(resourceName, SEGMENT_FILTER, `tags anyOf ["both-filter-test"]`),
					testAccCheckViewLinksAPIState(projectKey, "test-view", []string{"both-filter-flag-1"}),
					testAccCheckViewLinksSegmentsAPIState(projectKey, "test-view", []string{"both-filter-segment-1"}),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{FLAG_FILTER, SEGMENT_FILTER, SEGMENT_FILTER_ENVIRONMENT_ID, RECONCILE_ON_APPLY, RESOLVED_AT},
			},
			{
				// Remove segment_filter, keep flag_filter
				Config: fmt.Sprintf(testAccViewFilterLinksRemoveOneFilter, projectName, projectKey, maintainerID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewForFilterLinksExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, FLAG_FILTER, "tags:both-filter-test"),
					// Segments should be unlinked after removing the segment_filter
					testAccCheckViewLinksSegmentsAPIState(projectKey, "test-view", []string{}),
					// Flags should still be linked
					testAccCheckViewLinksAPIState(projectKey, "test-view", []string{"both-filter-flag-1"}),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{FLAG_FILTER, SEGMENT_FILTER, SEGMENT_FILTER_ENVIRONMENT_ID, RECONCILE_ON_APPLY, RESOLVED_AT},
			},
		},
	})
}

func TestAccViewFilterLinks_Triggers(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectName := "view-filter-links-trigger-test-" + projectKey
	resourceName := "launchdarkly_view_filter_links.test"

	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)

	members, _, err := client.ld.AccountMembersApi.GetMembers(client.ctx).Execute()
	require.NoError(t, err)
	require.True(t, len(members.Items) > 0, "This test requires at least one member in the account")

	maintainerID := members.Items[0].Id

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Step 1: Both flags tagged "trigger-test" → both should be linked
				Config:             fmt.Sprintf(testAccViewFilterLinksTriggersStep1, projectName, projectKey, maintainerID),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewForFilterLinksExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, FLAG_FILTER, "tags:trigger-test"),
					resource.TestCheckResourceAttr(resourceName, RECONCILE_ON_APPLY, "true"),
					testAccCheckViewLinksAPIState(projectKey, "test-view", []string{"trigger-flag-1", "trigger-flag-2"}),
				),
			},
			{
				// Step 2: trigger-flag-2 tag changed to "other-tag" (filter unchanged) → only trigger-flag-1 linked
				Config:             fmt.Sprintf(testAccViewFilterLinksTriggersStep2, projectName, projectKey, maintainerID),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewForFilterLinksExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, FLAG_FILTER, "tags:trigger-test"),
					resource.TestCheckResourceAttr(resourceName, RECONCILE_ON_APPLY, "true"),
					testAccCheckViewLinksAPIState(projectKey, "test-view", []string{"trigger-flag-1"}),
				),
			},
		},
	})
}

func TestAccViewFilterLinks_ReconcileOnApplyToggle(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectName := "view-filter-links-toggle-test-" + projectKey
	resourceName := "launchdarkly_view_filter_links.test"

	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)

	members, _, err := client.ld.AccountMembersApi.GetMembers(client.ctx).Execute()
	require.NoError(t, err)
	require.True(t, len(members.Items) > 0, "This test requires at least one member in the account")

	maintainerID := members.Items[0].Id

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Step 1: baseline (default behavior) should converge to an empty plan.
				Config: fmt.Sprintf(testAccViewFilterLinksToggleStep1, projectName, projectKey, maintainerID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewForFilterLinksExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, FLAG_FILTER, "tags:toggle-test"),
					resource.TestCheckResourceAttr(resourceName, RECONCILE_ON_APPLY, "false"),
					testAccCheckViewLinksAPIState(projectKey, "test-view", []string{"toggle-flag-1", "toggle-flag-2"}),
				),
			},
			{
				// Step 2: enabling reconcile_on_apply should intentionally produce a non-empty post-apply plan.
				Config:             fmt.Sprintf(testAccViewFilterLinksToggleStep2, projectName, projectKey, maintainerID),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewForFilterLinksExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, FLAG_FILTER, "tags:toggle-test"),
					resource.TestCheckResourceAttr(resourceName, RECONCILE_ON_APPLY, "true"),
					testAccCheckViewLinksAPIState(projectKey, "test-view", []string{"toggle-flag-1", "toggle-flag-2"}),
				),
			},
			{
				// Step 3: disabling reconcile_on_apply again should return to clean-plan behavior.
				Config: fmt.Sprintf(testAccViewFilterLinksToggleStep1, projectName, projectKey, maintainerID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewForFilterLinksExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, FLAG_FILTER, "tags:toggle-test"),
					resource.TestCheckResourceAttr(resourceName, RECONCILE_ON_APPLY, "false"),
					testAccCheckViewLinksAPIState(projectKey, "test-view", []string{"toggle-flag-1", "toggle-flag-2"}),
				),
			},
		},
	})
}

func TestAccViewFilterLinks_SegmentTriggers(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectName := "view-filter-links-segment-trigger-test-" + projectKey
	resourceName := "launchdarkly_view_filter_links.test"

	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)

	members, _, err := client.ld.AccountMembersApi.GetMembers(client.ctx).Execute()
	require.NoError(t, err)
	require.True(t, len(members.Items) > 0, "This test requires at least one member in the account")

	maintainerID := members.Items[0].Id

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Step 1: both segments match the filter and should be linked.
				Config:             fmt.Sprintf(testAccViewFilterLinksSegmentTriggersStep1, projectName, projectKey, maintainerID),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewForFilterLinksExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, SEGMENT_FILTER, `tags anyOf ["segment-trigger-test"]`),
					resource.TestCheckResourceAttr(resourceName, RECONCILE_ON_APPLY, "true"),
					testAccCheckViewLinksSegmentsAPIState(projectKey, "test-view", []string{"segment-trigger-1", "segment-trigger-2"}),
				),
			},
			{
				// Step 2: unchanged segment_filter, but segment2 tag changes away from the filter.
				// reconcile_on_apply should re-resolve and leave only segment-trigger-1 linked.
				Config:             fmt.Sprintf(testAccViewFilterLinksSegmentTriggersStep2, projectName, projectKey, maintainerID),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewForFilterLinksExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, SEGMENT_FILTER, `tags anyOf ["segment-trigger-test"]`),
					resource.TestCheckResourceAttr(resourceName, RECONCILE_ON_APPLY, "true"),
					testAccCheckViewLinksSegmentsAPIState(projectKey, "test-view", []string{"segment-trigger-1"}),
				),
			},
		},
	})
}

func testAccCheckViewForFilterLinksExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("view filter links ID is not set")
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
