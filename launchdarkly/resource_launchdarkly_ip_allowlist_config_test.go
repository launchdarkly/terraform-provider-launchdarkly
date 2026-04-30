package launchdarkly

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	testAccIpAllowlistConfigDefaults = `
resource "launchdarkly_ip_allowlist_config" "test" {
}
`

	testAccIpAllowlistConfigSessionEnabled = `
resource "launchdarkly_ip_allowlist_config" "test" {
	session_allowlist_enabled = true
}
`

	testAccIpAllowlistConfigBothEnabled = `
resource "launchdarkly_ip_allowlist_config" "test" {
	session_allowlist_enabled = true
	scoped_allowlist_enabled  = true
}
`

	testAccIpAllowlistConfigBothDisabled = `
resource "launchdarkly_ip_allowlist_config" "test" {
	session_allowlist_enabled = false
	scoped_allowlist_enabled  = false
}
`

	testAccIpAllowlistConfigDuplicateFirst = `
resource "launchdarkly_ip_allowlist_config" "first" {
	session_allowlist_enabled = true
}
`

	testAccIpAllowlistConfigDuplicateSecond = `
resource "launchdarkly_ip_allowlist_config" "first" {
	session_allowlist_enabled = true
}

resource "launchdarkly_ip_allowlist_config" "second" {
	scoped_allowlist_enabled = true
}
`
)

func TestAccIpAllowlistConfig(t *testing.T) {
	resourceName := "launchdarkly_ip_allowlist_config.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccIpAllowlistConfigDefaults,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIpAllowlistConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, SESSION_ALLOWLIST_ENABLED, "false"),
					resource.TestCheckResourceAttr(resourceName, SCOPED_ALLOWLIST_ENABLED, "false"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     ipAllowlistConfigID,
				ImportStateVerify: true,
			},
			{
				Config: testAccIpAllowlistConfigSessionEnabled,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIpAllowlistConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, SESSION_ALLOWLIST_ENABLED, "true"),
					resource.TestCheckResourceAttr(resourceName, SCOPED_ALLOWLIST_ENABLED, "false"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     ipAllowlistConfigID,
				ImportStateVerify: true,
			},
			{
				Config: testAccIpAllowlistConfigBothDisabled,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIpAllowlistConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, SESSION_ALLOWLIST_ENABLED, "false"),
					resource.TestCheckResourceAttr(resourceName, SCOPED_ALLOWLIST_ENABLED, "false"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     ipAllowlistConfigID,
				ImportStateVerify: true,
			},
			{
				Config: testAccIpAllowlistConfigBothEnabled,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIpAllowlistConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, SESSION_ALLOWLIST_ENABLED, "true"),
					resource.TestCheckResourceAttr(resourceName, SCOPED_ALLOWLIST_ENABLED, "true"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     ipAllowlistConfigID,
				ImportStateVerify: true,
			},
			{
				Config: testAccIpAllowlistConfigDefaults,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIpAllowlistConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, SESSION_ALLOWLIST_ENABLED, "false"),
					resource.TestCheckResourceAttr(resourceName, SCOPED_ALLOWLIST_ENABLED, "false"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     ipAllowlistConfigID,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccIpAllowlistConfig_DuplicateErrors(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccIpAllowlistConfigDuplicateFirst,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIpAllowlistConfigExists("launchdarkly_ip_allowlist_config.first"),
					resource.TestCheckResourceAttr("launchdarkly_ip_allowlist_config.first", SESSION_ALLOWLIST_ENABLED, "true"),
				),
			},
			{
				Config:      testAccIpAllowlistConfigDuplicateSecond,
				ExpectError: regexp.MustCompile(`Only one launchdarkly_ip_allowlist_config resource should exist per account`),
			},
		},
	})
}

func testAccCheckIpAllowlistConfigExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("IP allowlist config ID is not set")
		}
		client := testAccProvider.Meta().(*Client)
		_, err := getIpAllowlist(client)
		if err != nil {
			return fmt.Errorf("error getting IP allowlist config: %s", err)
		}
		return nil
	}
}
