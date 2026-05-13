package launchdarkly

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// testAccIpAllowlistEntryTestIPs are the IPs / CIDRs the acceptance
// tests in this file create. The LD account allowlist is a single
// shared document; if a previous CI run's Create succeeded but a
// later test step failed before the framework recorded state, the
// orphan entry survives and poisons the next run with a 409
// optimistic_locking_error (LD's API reuses the code for both
// genuine version races and duplicate-IP rejections).
//
// cleanupOrphanIpAllowlistEntries runs as part of each test's
// PreCheck and DELETEs any entry whose ipAddress matches one of
// these tests' targets. Best-effort: a missing entry is success,
// transient errors are logged and ignored so the test still gets
// a chance to surface a real-account problem.
var testAccIpAllowlistEntryTestIPs = []string{"52.1.1.1", "54.0.0.0/24"}

func cleanupOrphanIpAllowlistEntries(t *testing.T) {
	t.Helper()
	// Build a client directly from env: testAccProvider.Meta() is nil
	// until terraform-plugin-sdk configures the provider, which only
	// happens once a test Step runs. PreCheck fires before that.
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		t.Logf("ip-allowlist cleanup: client construction failed (continuing): %s", err)
		return
	}
	allowlist, err := getIpAllowlist(client)
	if err != nil {
		t.Logf("ip-allowlist cleanup probe failed (continuing): %s", err)
		return
	}
	targets := map[string]struct{}{}
	for _, ip := range testAccIpAllowlistEntryTestIPs {
		targets[ip] = struct{}{}
	}
	for _, entry := range allowlist.Entries {
		if _, hit := targets[entry.IpAddress]; !hit {
			continue
		}
		if delErr := deleteIpAllowlistEntry(client, entry.ID); delErr != nil {
			t.Logf("ip-allowlist cleanup: failed to delete orphan %s (%s): %s", entry.ID, entry.IpAddress, delErr)
			continue
		}
		t.Logf("ip-allowlist cleanup: deleted orphan entry %s for %s", entry.ID, entry.IpAddress)
	}
}

const (
	testAccIpAllowlistEntryCreate = `
resource "launchdarkly_ip_allowlist_entry" "test" {
	ip_address  = "52.1.1.1"
	description = "Test IP allowlist entry"
}
`

	testAccIpAllowlistEntryUpdateDescription = `
resource "launchdarkly_ip_allowlist_entry" "test" {
	ip_address  = "52.1.1.1"
	description = "Updated description"
}
`

	testAccIpAllowlistEntryNoDescription = `
resource "launchdarkly_ip_allowlist_entry" "test" {
	ip_address = "52.1.1.1"
}
`

	testAccIpAllowlistEntryCIDR = `
resource "launchdarkly_ip_allowlist_entry" "cidr" {
	ip_address  = "54.0.0.0/24"
	description = "CIDR block entry"
}
`
)

func TestAccIpAllowlistEntry_CreateAndUpdate(t *testing.T) {
	resourceName := "launchdarkly_ip_allowlist_entry.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			cleanupOrphanIpAllowlistEntries(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccIpAllowlistEntryCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIpAllowlistEntryExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, IP_ADDRESS, "52.1.1.1"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "Test IP allowlist entry"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccIpAllowlistEntryCreate, // should not change
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIpAllowlistEntryExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, IP_ADDRESS, "52.1.1.1"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "Test IP allowlist entry"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccIpAllowlistEntryUpdateDescription,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIpAllowlistEntryExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, IP_ADDRESS, "52.1.1.1"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "Updated description"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccIpAllowlistEntryNoDescription,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIpAllowlistEntryExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, IP_ADDRESS, "52.1.1.1"),
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

func TestAccIpAllowlistEntry_CIDRBlock(t *testing.T) {
	resourceName := "launchdarkly_ip_allowlist_entry.cidr"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			cleanupOrphanIpAllowlistEntries(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccIpAllowlistEntryCIDR,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIpAllowlistEntryExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, IP_ADDRESS, "54.0.0.0/24"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "CIDR block entry"),
				),
			},
		},
	})
}

func testAccCheckIpAllowlistEntryExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("IP allowlist entry ID is not set")
		}
		client := testAccProvider.Meta().(*Client)
		allowlist, err := getIpAllowlist(client)
		if err != nil {
			return fmt.Errorf("error getting IP allowlist: %s", err)
		}
		if findIpAllowlistEntryByID(allowlist.Entries, rs.Primary.ID) == nil {
			return fmt.Errorf("IP allowlist entry %s not found", rs.Primary.ID)
		}
		return nil
	}
}
