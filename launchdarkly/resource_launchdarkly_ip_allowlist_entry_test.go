package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

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
