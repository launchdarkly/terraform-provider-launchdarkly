package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	ldapi "github.com/launchdarkly/api-client-go"
)

const (
	testAccCustomRoleCreate = `
resource "launchdarkly_custom_role" "test" {
	key = "custom-role-key-1"
	name = "custom-role-name-1"
	description= "crd"
	policy = [
	{
		actions = ["*"]	
		effect = "allow"
		resources = ["proj/*:env/production"]
	}
	]
}
`
	testAccCustomRoleUpdate = `
resource "launchdarkly_custom_role" "test" {
	key = "custom-role-key-1"
	name = "Custom role - deny production"
	description= "Deny all actions on production environments"
	policy = [
	{
		actions = ["*"]	
		effect = "deny"
		resources = ["proj/*:env/production"]
	}
	]
}
`
)

func TestAccCustomRole_Create(t *testing.T) {
	resourceName := "launchdarkly_custom_role.test"
	policy := ldapi.Policy{
		Resources: []string{"proj/*:env/production"},
		Actions:   []string{"*"},
		Effect:    "allow",
	}
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCustomRoleCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", "custom-role-key-1"),
					resource.TestCheckResourceAttr(resourceName, "name", "custom-role-name-1"),
					resource.TestCheckResourceAttr(resourceName, "description", "crd"),
					resource.TestCheckResourceAttr(resourceName, "policy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, testAccPolicyKey(policy, "actions.#"), "1"),
					resource.TestCheckResourceAttr(resourceName, testAccPolicyKey(policy, "actions.0"), "*"),
					resource.TestCheckResourceAttr(resourceName, testAccPolicyKey(policy, "resources.#"), "1"),
					resource.TestCheckResourceAttr(resourceName, testAccPolicyKey(policy, "resources.0"), "proj/*:env/production"),
					resource.TestCheckResourceAttr(resourceName, testAccPolicyKey(policy, effect), "allow"),
				),
			},
		},
	})
}

func TestAccCustomRole_Update(t *testing.T) {
	resourceName := "launchdarkly_custom_role.test"
	policy := ldapi.Policy{
		Resources: []string{"proj/*:env/production"},
		Actions:   []string{"*"},
		Effect:    "deny",
	}
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCustomRoleCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(resourceName),
				),
			},
			{
				Config: testAccCustomRoleUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "key", "custom-role-key-1"),
					resource.TestCheckResourceAttr(resourceName, "name", "Custom role - deny production"),
					resource.TestCheckResourceAttr(resourceName, "description", "Deny all actions on production environments"),
					resource.TestCheckResourceAttr(resourceName, "policy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, testAccPolicyKey(policy, "actions.#"), "1"),
					resource.TestCheckResourceAttr(resourceName, testAccPolicyKey(policy, "actions.0"), "*"),
					resource.TestCheckResourceAttr(resourceName, testAccPolicyKey(policy, "resources.#"), "1"),
					resource.TestCheckResourceAttr(resourceName, testAccPolicyKey(policy, "resources.0"), "proj/*:env/production"),
					resource.TestCheckResourceAttr(resourceName, testAccPolicyKey(policy, effect), "deny"),
				),
			},
		},
	})
}

func testAccPolicyKey(policy ldapi.Policy, subkey string) string {
	return fmt.Sprintf("policy.%d.%s", hashcode.String(fmt.Sprintf("%v", policy)), subkey)
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
