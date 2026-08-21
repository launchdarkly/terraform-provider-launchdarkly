package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	testAccCustomRoleCreate = `
	resource "launchdarkly_custom_role" "test" {
		key = "%s"
		name = "Custom role - %s"
		description = "Deny all actions on production environments"
		base_permissions = "no_access"
		policy {
			actions = ["*"]	
			effect = "deny"
			resources = ["proj/*:env/production"]
		}
	}
`
	// IMPORTANT TO NOTE that the $ character must be escaped in terraform by using a double $$
	// otherwas ${} will be interpreted as a terraform variable and throw an error
	testAccCustomRoleUpdate = `
resource "launchdarkly_custom_role" "test" {
	key = "%s"
	name = "Updated - %s"
	policy {
		actions = ["*"]	
		effect = "allow"
		resources = ["proj/*:env/$${roleAttribute/devEnvironments}"]
	}
}
`
	testAccCustomRoleCreateWithStatements = `
resource "launchdarkly_custom_role" "test" {
	key = "%s"
	name = "Custom role - %s"
	description = "Allow all actions on staging environments"
	policy_statements {
		actions = ["*"]	
		effect = "allow"
		resources = ["proj/$${roleAttribute/devProjects}:env/staging"]
	}
}
`
	testAccCustomRoleUpdateWithStatements = `
resource "launchdarkly_custom_role" "test" {
	key = "%s"
	name = "Updated role - %s"
	description= "Deny all actions on production environments"
	policy_statements {
		actions = ["*"]	
		effect = "deny"
		resources = ["proj/*:env/production"]
	}
}
`
	testAccCustomRoleCreateWithNotStatements = `
resource "launchdarkly_custom_role" "test" {
	key = "%s"
	name = "Custom role - %s"
	description = "Don't allow all actions on non-staging environments"
	policy_statements {
		not_actions = ["*"]	
		effect = "allow"
		not_resources = ["proj/*:env/staging"]
	}
}
`
	testAccCustomRoleUpdateWithNotStatements = `
resource "launchdarkly_custom_role" "test" {
	key = "%s"
	name = "Updated role - %s"
	description= "Don't deny all actions on non production environments"
	policy_statements {
		not_actions = ["*"]
		effect = "deny"
		not_resources = ["proj/*:env/production"]
	}
}
`
	testAccCustomRoleAssignedToTeam = `
resource "launchdarkly_custom_role" "delete_test" {
	key = "%s"
	name = "Delete ordering role - %s"
	policy_statements {
		actions = ["*"]
		effect = "allow"
		resources = ["proj/*:env/staging"]
	}
}

resource "launchdarkly_team" "delete_test" {
	key = "%s"
	name = "delete ordering team"
	description = "REL-12313 role deletion interdependency test"
	custom_role_keys = [launchdarkly_custom_role.delete_test.key]
}
`
	testAccCustomRoleUnassignedFromTeam = `
resource "launchdarkly_team" "delete_test" {
	key = "%s"
	name = "delete ordering team"
	description = "REL-12313 role deletion interdependency test"
	custom_role_keys = []
}
`
)

func TestAccCustomRole_CreateAndUpdate(t *testing.T) {
	key := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_custom_role.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCustomRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCustomRoleCreate, key, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, key),
					resource.TestCheckResourceAttr(resourceName, NAME, "Custom role - "+name),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "Deny all actions on production environments"),
					resource.TestCheckResourceAttr(resourceName, BASE_PERMISSIONS, "no_access"),
					resource.TestCheckResourceAttr(resourceName, "policy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.resources.0", "proj/*:env/production"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.effect", "deny"),
				),
			},
			{
				Config: fmt.Sprintf(testAccCustomRoleUpdate, key, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, key),
					resource.TestCheckResourceAttr(resourceName, NAME, "Updated - "+name),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, ""), // should be empty after removal
					resource.TestCheckResourceAttr(resourceName, BASE_PERMISSIONS, "reader"),
					resource.TestCheckResourceAttr(resourceName, "policy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.resources.0", "proj/*:env/${roleAttribute/devEnvironments}"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.effect", "allow"),
				),
			},
		},
	})
}

func TestAccCustomRole_CreateAndUpdateWithStatements(t *testing.T) {
	key := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_custom_role.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCustomRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCustomRoleCreateWithStatements, key, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, key),
					resource.TestCheckResourceAttr(resourceName, NAME, "Custom role - "+name),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "Allow all actions on staging environments"),
					resource.TestCheckResourceAttr(resourceName, "policy.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.resources.0", "proj/${roleAttribute/devProjects}:env/staging"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.effect", "allow"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccCustomRoleUpdateWithStatements, key, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, key),
					resource.TestCheckResourceAttr(resourceName, NAME, "Updated role - "+name),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "Deny all actions on production environments"),
					resource.TestCheckResourceAttr(resourceName, "policy.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.resources.0", "proj/*:env/production"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.effect", "deny"),
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

func TestAccCustomRole_CreateAndUpdateWithNotStatements(t *testing.T) {
	key := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_custom_role.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCustomRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCustomRoleCreateWithNotStatements, key, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, key),
					resource.TestCheckResourceAttr(resourceName, NAME, "Custom role - "+name),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "Don't allow all actions on non-staging environments"),
					resource.TestCheckResourceAttr(resourceName, BASE_PERMISSIONS, "reader"),
					resource.TestCheckResourceAttr(resourceName, "policy.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.not_actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.not_actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.not_resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.not_resources.0", "proj/*:env/staging"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.effect", "allow"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccCustomRoleUpdateWithNotStatements, key, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, key),
					resource.TestCheckResourceAttr(resourceName, NAME, "Updated role - "+name),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "Don't deny all actions on non production environments"),
					resource.TestCheckResourceAttr(resourceName, "policy.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.not_actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.not_actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.not_resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.not_resources.0", "proj/*:env/production"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.effect", "deny"),
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

func testAccCheckCustomRoleExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("custom role ID is not set")
		}
		client := testAccProvider.Meta().(*Client)
		_, _, err := client.ld.CustomRolesApi.GetCustomRole(client.ctx, rs.Primary.ID).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting custom role. %s", err)
		}
		return nil
	}
}

// TestAccCustomRole_DeleteWhileAssignedToTeam covers REL-12313: an apply
// that deletes a custom role and, in the same apply, removes it from a
// team's custom_role_keys. Terraform only orders operations by references
// in the new configuration, so the role DELETE can fire before the team
// update lands and LaunchDarkly rejects it with a 409. The provider must
// retry the deletion until the unassignment completes.
func TestAccCustomRole_DeleteWhileAssignedToTeam(t *testing.T) {
	roleKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	roleName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	teamKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_custom_role.delete_test"
	teamResourceName := "launchdarkly_team.delete_test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCustomRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCustomRoleAssignedToTeam, roleKey, roleName, teamKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(resourceName),
					resource.TestCheckResourceAttr(teamResourceName, "custom_role_keys.#", "1"),
				),
			},
			{
				Config: fmt.Sprintf(testAccCustomRoleUnassignedFromTeam, teamKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(teamResourceName, "custom_role_keys.#", "0"),
					// The role is out of state after this step, so
					// testAccCheckCustomRoleDestroy would pass vacuously;
					// assert server-side deletion by key instead.
					func(_ *terraform.State) error {
						client := testAccProvider.Meta().(*Client)
						_, res, err := client.ld.CustomRolesApi.GetCustomRole(client.ctx, roleKey).Execute()
						if isStatusNotFound(res) {
							return nil
						}
						if err != nil {
							return err
						}
						return fmt.Errorf("custom role %s still exists after delete-while-assigned apply", roleKey)
					},
				),
			},
		},
	})
}

// testAccCheckCustomRoleDestroy verifies the custom role has been destroyed
func testAccCheckCustomRoleDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_custom_role" {
			continue
		}

		_, res, err := client.ld.CustomRolesApi.GetCustomRole(client.ctx, rs.Primary.ID).Execute()

		if isStatusNotFound(res) {
			continue
		}

		if err != nil {
			return err
		}

		return fmt.Errorf("custom role %s still exists", rs.Primary.ID)
	}
	return nil
}
