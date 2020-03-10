package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
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

func TestAccCustomRole_Create(t *testing.T) {
	key := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
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

func TestAccCustomRole_Update(t *testing.T) {
	key := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_custom_role.test"
	policy := ldapi.Policy{
		Resources: []string{"proj/*:env/staging"},
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
