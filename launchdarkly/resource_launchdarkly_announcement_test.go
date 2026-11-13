package launchdarkly

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	// An announcement is an account-scoped SINGLETON (the API allows only one per
	// account), so a leftover from a prior aborted run would 409 this test's create.
	// testAccAnnouncementPreClean clears the slot first. The fixtures carry this
	// marker only to label the artifacts as test-owned in the dedicated Terraform
	// acceptance account; the pre-clean deletes any leftover regardless.
	tfAccAnnouncementSentinel = "tf-acc-test announcement"

	testAccAnnouncementTitle       = tfAccAnnouncementSentinel + " (safe to delete)"
	testAccAnnouncementUpdateTitle = tfAccAnnouncementSentinel + " (updated, safe to delete)"

	// start_time / end_time are Unix timestamps in milliseconds. These are
	// well in the future so the fixtures don't depend on wall-clock time.
	testAccAnnouncementStartTime = "1893456000000" // 2030-01-01T00:00:00Z
	testAccAnnouncementEndTime   = "1924992000000" // 2031-01-01T00:00:00Z
)

var (
	testAccAnnouncementCreate = fmt.Sprintf(`
resource "launchdarkly_announcement" "test" {
	title          = %q
	message        = "We will perform scheduled maintenance soon."
	severity       = "warning"
	is_dismissible = true
	start_time     = %s
}
`, testAccAnnouncementTitle, testAccAnnouncementStartTime)

	testAccAnnouncementUpdate = fmt.Sprintf(`
resource "launchdarkly_announcement" "test" {
	title          = %q
	message        = "Maintenance has been rescheduled. Thank you for your patience."
	severity       = "warning"
	is_dismissible = false
	start_time     = %s
	end_time       = %s
}
`, testAccAnnouncementUpdateTitle, testAccAnnouncementStartTime, testAccAnnouncementEndTime)

	testAccAnnouncementInvalidSeverity = fmt.Sprintf(`
resource "launchdarkly_announcement" "test" {
	title          = %q
	message        = "This should fail validation."
	severity       = "not-a-severity"
	is_dismissible = true
	start_time     = %s
}
`, tfAccAnnouncementSentinel+" (invalid, safe to delete)", testAccAnnouncementStartTime)
)

func TestAccAnnouncement_Create(t *testing.T) {
	resourceName := "launchdarkly_announcement.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAnnouncementPreClean(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccAnnouncementDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAnnouncementCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAnnouncementExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, TITLE, testAccAnnouncementTitle),
					resource.TestCheckResourceAttr(resourceName, MESSAGE, "We will perform scheduled maintenance soon."),
					resource.TestCheckResourceAttr(resourceName, SEVERITY, "warning"),
					resource.TestCheckResourceAttr(resourceName, IS_DISMISSIBLE, "true"),
					resource.TestCheckResourceAttr(resourceName, START_TIME, testAccAnnouncementStartTime),
					resource.TestCheckResourceAttrSet(resourceName, ID),
					resource.TestCheckResourceAttrSet(resourceName, STATUS),
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

func TestAccAnnouncement_Update(t *testing.T) {
	resourceName := "launchdarkly_announcement.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAnnouncementPreClean(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccAnnouncementDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAnnouncementCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAnnouncementExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, TITLE, testAccAnnouncementTitle),
					resource.TestCheckResourceAttr(resourceName, IS_DISMISSIBLE, "true"),
					resource.TestCheckNoResourceAttr(resourceName, END_TIME),
				),
			},
			{
				Config: testAccAnnouncementUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAnnouncementExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, TITLE, testAccAnnouncementUpdateTitle),
					resource.TestCheckResourceAttr(resourceName, MESSAGE, "Maintenance has been rescheduled. Thank you for your patience."),
					resource.TestCheckResourceAttr(resourceName, IS_DISMISSIBLE, "false"),
					resource.TestCheckResourceAttr(resourceName, END_TIME, testAccAnnouncementEndTime),
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

func TestAccAnnouncement_InvalidSeverity(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccAnnouncementInvalidSeverity,
				ExpectError: regexp.MustCompile(`(?i)Attribute severity value must be one of`),
			},
		},
	})
}

func testAccCheckAnnouncementExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("announcement ID is not set")
		}
		found, err := announcementExistsByID(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("received an error getting announcement: %s", err)
		}
		if !found {
			return fmt.Errorf("announcement %q not found", rs.Primary.ID)
		}
		return nil
	}
}

func testAccAnnouncementDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_announcement" {
			continue
		}
		found, err := announcementExistsByID(rs.Primary.ID)
		if err != nil {
			// A non-404 error must fail the check rather than pass silently.
			return fmt.Errorf("error checking whether announcement %q was destroyed: %s", rs.Primary.ID, err)
		}
		if found {
			return fmt.Errorf("announcement %q still exists", rs.Primary.ID)
		}
	}
	return nil
}

// announcementExistsByID resolves an announcement through the list endpoint,
// mirroring the resource's own read path (there is no GET-by-ID endpoint).
func announcementExistsByID(id string) (bool, error) {
	client := mustTestAccClient()
	var offset int32
	for {
		page, _, err := client.ld.AnnouncementsApi.GetAnnouncementsPublic(client.ctx).
			Limit(announcementListPageSize).
			Offset(offset).
			Execute()
		if err != nil {
			return false, handleLdapiErr(err)
		}
		for i := range page.Items {
			if page.Items[i].Id == id {
				return true, nil
			}
		}
		if len(page.Items) < announcementListPageSize {
			return false, nil
		}
		offset += announcementListPageSize
	}
}

// testAccAnnouncementPreClean clears any pre-existing announcement so this test's
// one-per-account create does not 409. LAUNCHDARKLY_ACCESS_TOKEN points at a
// dedicated Terraform acceptance-testing account (not a customer/prod account), so
// any leftover — e.g. from an aborted prior run or a retained verification run — is
// a test artifact safe to delete. Mirrors the account-singleton orphan-cleanup
// PreCheck pattern. IDs are collected before deleting so paging isn't disturbed
// mid-iteration (in practice there is at most one, given the singleton limit).
func testAccAnnouncementPreClean(t *testing.T) {
	client := mustTestAccClient()
	var (
		ids    []string
		offset int32
	)
	for {
		page, _, err := client.ld.AnnouncementsApi.GetAnnouncementsPublic(client.ctx).
			Limit(announcementListPageSize).
			Offset(offset).
			Execute()
		if err != nil {
			t.Fatalf("announcement pre-clean: list failed: %s", handleLdapiErr(err))
		}
		for i := range page.Items {
			ids = append(ids, page.Items[i].Id)
		}
		if len(page.Items) < announcementListPageSize {
			break
		}
		offset += announcementListPageSize
	}
	for _, id := range ids {
		if _, err := client.ld.AnnouncementsApi.DeleteAnnouncementPublic(client.ctx, id).Execute(); err != nil {
			t.Fatalf("announcement pre-clean: delete %q failed: %s", id, handleLdapiErr(err))
		}
	}
}
