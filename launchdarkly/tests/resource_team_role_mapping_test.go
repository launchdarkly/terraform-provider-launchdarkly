package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// testAccTeamRoleMappingSetup scaffolds the roles and team necessary for the team/role mapping resource
func testAccTeamRoleMappingSetup(uniqueRole0, uniqueRole1, teamKey string) string {
	return fmt.Sprintf(`
	resource "launchdarkly_custom_role" "role_0" {
		key              = "%s"
		name             = "Custom Role 1 %s"
		base_permissions = "no_access"
		policy {
			actions = ["*"]	
			effect = "deny"
			resources = ["proj/*:env/production"]
		}
	}

	resource "launchdarkly_custom_role" "role_1" {
		key              = "%s"
		name             = "Custom Role 2 %s"
		base_permissions = "no_access"
		policy {
			actions = ["*"]	
			effect = "deny"
			resources = ["proj/*:env/test"]
		}
	}

	resource "launchdarkly_team" "test_team" {
		key  = "%s"
		name = "Test Team"
    member_ids  = []
    maintainers = []

		# custom_role_keys is empty here because we are using the mapping resource
    custom_role_keys = []

    lifecycle {
      ignore_changes = [
        # Ignore changes custom_role_keys because we are using the mapping resource
        custom_role_keys
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

func TestAccTeamRoleMapping_basic(t *testing.T) {
	t.Parallel()
	resourceName := "launchdarkly_team_role_mapping.basic"
	role0 := "dummy-role-0-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	role1 := "dummy-role-1-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	teamKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccFrameworkMuxProviders(context.Background(), t),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccBasicTeamRoleMappingConfig(role0, role1, teamKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "team_key", teamKey),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.0", role0),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.1", role1),
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
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.0", role1),
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
		ProtoV5ProviderFactories: testAccFrameworkMuxProviders(context.Background(), t),
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
