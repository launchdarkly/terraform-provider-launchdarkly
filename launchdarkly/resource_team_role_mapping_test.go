package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// testAccTeamRoleMappingSetup scaffolds the roles and team necessary for the team/role mapping resource
func testAccTeamRoleMappingSetup(uniqueRole0, uniqueRole1, teamKey string) string {
	return fmt.Sprintf(`
	resource "launchdarkly_custom_role" "role_0" {
		key              = "%s"
		name             = "Custom Role 1 %s"
		base_permissions = "no_access"
		policy_statements {
			actions   = ["*"]
			effect    = "deny"
			resources = ["proj/*:env/production"]
		}
	}

	resource "launchdarkly_custom_role" "role_1" {
		key              = "%s"
		name             = "Custom Role 2 %s"
		base_permissions = "no_access"
		policy_statements {
			actions   = ["*"]
			effect    = "deny"
			resources = ["proj/*:env/test"]
		}
	}

	resource "launchdarkly_team" "test_team" {
		key         = "%s"
		name        = "Test Team"
		member_ids  = []
		maintainers = []

		# custom_role_keys is empty here because we are using the mapping resource
		custom_role_keys = []

		lifecycle {
			ignore_changes = [
				# Ignore changes to custom_role_keys and role_attributes because we are using the mapping resource
				custom_role_keys,
				role_attributes,
			]
		}

		# Use depends_on to ensure the team gets deleted before the roles because the LD API
		# prevents deleting custom roles that are still in use by teams.
		depends_on = [launchdarkly_custom_role.role_0, launchdarkly_custom_role.role_1]
	}
	`, uniqueRole0, uniqueRole0, uniqueRole1, uniqueRole1, teamKey)
}

func testAccBasicTeamRoleMappingConfig(uniqueRole0, uniqueRole1, teamKey string) string {
	return fmt.Sprintf(`
	%s

	resource "launchdarkly_team_role_mapping" "basic" {
		team_key = launchdarkly_team.test_team.key

		custom_role_keys = [
			launchdarkly_custom_role.role_0.key,
			launchdarkly_custom_role.role_1.key
		]
	}
	`, testAccTeamRoleMappingSetup(uniqueRole0, uniqueRole1, teamKey))
}

func testAccBasicTeamRoleMappingConfigUpdate(uniqueRole0, uniqueRole1, teamKey string) string {
	return fmt.Sprintf(`
	%s

	resource "launchdarkly_team_role_mapping" "basic" {
		team_key = launchdarkly_team.test_team.key

		custom_role_keys = [
			launchdarkly_custom_role.role_1.key,
		]
	}
	`, testAccTeamRoleMappingSetup(uniqueRole0, uniqueRole1, teamKey))
}

func testAccBasicTeamRoleMappingConfigEmpty(uniqueRole0, uniqueRole1, teamKey string) string {
	return fmt.Sprintf(`
	%s

	resource "launchdarkly_team_role_mapping" "basic" {
		team_key = launchdarkly_team.test_team.key

		custom_role_keys = []
	}
	`, testAccTeamRoleMappingSetup(uniqueRole0, uniqueRole1, teamKey))
}

// launchdarkly_team_role_mapping is a plugin-framework resource, so these tests
// need the muxed provider (testAccProtoV5ProviderFactories) rather than the
// SDKv2-only testAccProviders map every other acceptance test in this package
// uses.
//
// custom_role_keys is a set, so membership is asserted with
// TestCheckTypeSetElemAttr rather than by index — set element order in state is
// not meaningful and indexed assertions would be a false failure waiting to
// happen.
func TestAccTeamRoleMapping_basic(t *testing.T) {
	t.Parallel()
	resourceName := "launchdarkly_team_role_mapping.basic"
	role0 := "dummy-role-0-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	role1 := "dummy-role-1-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	teamKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccBasicTeamRoleMappingConfig(role0, role1, teamKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "team_key", teamKey),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "custom_role_keys.*", role0),
					resource.TestCheckTypeSetElemAttr(resourceName, "custom_role_keys.*", role1),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccBasicTeamRoleMappingConfigUpdate(role0, role1, teamKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "team_key", teamKey),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "custom_role_keys.*", role1),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccBasicTeamRoleMappingConfigEmpty(role0, role1, teamKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "team_key", teamKey),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.#", "0"),
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

func TestAccTeamRoleMapping_empty(t *testing.T) {
	resourceName := "launchdarkly_team_role_mapping.basic"
	role0 := "role-0-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	role1 := "role-1-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	teamKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccBasicTeamRoleMappingConfigEmpty(role0, role1, teamKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "team_key", teamKey),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.#", "0"),
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
