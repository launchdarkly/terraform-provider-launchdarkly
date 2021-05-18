package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	ldapi "github.com/launchdarkly/api-client-go"
)

func testAccCustomRoleCreate(randomKey, randomName string) string {
	return fmt.Sprintf(`
	resource "launchdarkly_custom_role" "test" {
		key = "%s"
		name = "Custom role - %s"
		description= "Deny all actions on production environments"
		policy {
			actions = ["*"]
			effect = "deny"
			resources = ["proj/*:env/production"]
		}
	}
	`, randomKey, randomName)
}

func testAccCustomRoleUpdate(randomKey, randomName string) string {
	return fmt.Sprintf(`resource "launchdarkly_custom_role" "test" {
	key = "%s"
	name = "Updated - %s"
	description= "Allow all actions on staging environments"
	policy {
		actions = ["*"]
		effect = "allow"
		resources = ["proj/*:env/staging"]
	}
}
`, randomKey, randomName)
}

func testAccCustomRoleCreateWithStatements(randomKey, randomName string) string {
	return fmt.Sprintf(`resource "launchdarkly_custom_role" "test" {
	key = "%s"
	name = "Custom role - %s"
	description= "Allow all actions on staging environments"
	policy_statements {
		actions = ["*"]
		effect = "allow"
		resources = ["proj/*:env/staging"]
	}
}
`, randomKey, randomName)
}

func testAccCustomRoleUpdateWithStatements(randomKey, randomName string) string {
	return fmt.Sprintf(`resource "launchdarkly_custom_role" "test" {
	key = "%s"
	name = "Updated role - %s"
	description= "Deny all actions on production environments"
	policy_statements {
		actions = ["*"]
		effect = "deny"
		resources = ["proj/*:env/production"]
	}
}
`, randomKey, randomName)
}

func TestAccCustomRole_Create(t *testing.T) {
	key := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_custom_role.test"
	policy := ldapi.Policy{
		Resources: []string{"proj/*:env/production"},
		Actions:   []string{"*"},
		Effect:    "deny",
	}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCustomRoleCreate(key, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", key),
					resource.TestCheckResourceAttr(resourceName, "name", "Custom role - "+name),
					resource.TestCheckResourceAttr(resourceName, "description", "Deny all actions on production environments"),
					resource.TestCheckResourceAttr(resourceName, "policy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, testAccPolicyKey(policy, "actions.#"), "1"),
					resource.TestCheckResourceAttr(resourceName, testAccPolicyKey(policy, "actions.0"), "*"),
					resource.TestCheckResourceAttr(resourceName, testAccPolicyKey(policy, "resources.#"), "1"),
					resource.TestCheckResourceAttr(resourceName, testAccPolicyKey(policy, "resources.0"), "proj/*:env/production"),
					resource.TestCheckResourceAttr(resourceName, testAccPolicyKey(policy, EFFECT), "deny"),
				),
			},
		},
	})
}

func TestAccCustomRole_CreateWithStatements(t *testing.T) {
	key := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_custom_role.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCustomRoleCreateWithStatements(key, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", key),
					resource.TestCheckResourceAttr(resourceName, "name", "Custom role - "+name),
					resource.TestCheckResourceAttr(resourceName, "description", "Allow all actions on staging environments"),
					resource.TestCheckResourceAttr(resourceName, "policy.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.resources.0", "proj/*:env/staging"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.effect", "allow"),
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

func TestAccCustomRole_Update(t *testing.T) {
	key := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_custom_role.test"
	policy := ldapi.Policy{
		Resources: []string{"proj/*:env/staging"},
		Actions:   []string{"*"},
		Effect:    "allow",
	}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCustomRoleCreate(key, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(resourceName),
				),
			},
			{
				Config: testAccCustomRoleUpdate(key, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", key),
					resource.TestCheckResourceAttr(resourceName, "name", "Updated - "+name),
					resource.TestCheckResourceAttr(resourceName, "description", "Allow all actions on staging environments"),
					resource.TestCheckResourceAttr(resourceName, "policy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, testAccPolicyKey(policy, "actions.#"), "1"),
					resource.TestCheckResourceAttr(resourceName, testAccPolicyKey(policy, "actions.0"), "*"),
					resource.TestCheckResourceAttr(resourceName, testAccPolicyKey(policy, "resources.#"), "1"),
					resource.TestCheckResourceAttr(resourceName, testAccPolicyKey(policy, "resources.0"), "proj/*:env/staging"),
					resource.TestCheckResourceAttr(resourceName, testAccPolicyKey(policy, EFFECT), "allow"),
				),
			},
		},
	})
}

func TestAccCustomRole_UpdateWithStatements(t *testing.T) {
	key := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_custom_role.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCustomRoleCreate(key, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(resourceName),
				),
			},
			{
				Config: testAccCustomRoleUpdateWithStatements(key, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", key),
					resource.TestCheckResourceAttr(resourceName, "name", "Updated role - "+name),
					resource.TestCheckResourceAttr(resourceName, "description", "Deny all actions on production environments"),
					resource.TestCheckResourceAttr(resourceName, "policy.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.resources.0", "proj/*:env/production"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.effect", "deny"),
				),
			},
		},
	})
}
func testAccPolicyKey(policy ldapi.Policy, subkey string) string {
	return fmt.Sprintf("policy.%d.%s", schema.HashString(fmt.Sprintf("%v", policy)), subkey)
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
		_, _, err := client.ld.CustomRolesApi.GetCustomRole(client.ctx, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("received an error getting custom role. %s", err)
		}
		return nil
	}
}
