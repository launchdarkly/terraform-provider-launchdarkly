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
		description= "Deny all actions on production environments"
		policy {
			actions = ["*"]	
			effect = "deny"
			resources = ["proj/*:env/production"]
		}
	}
`
	testAccCustomRoleUpdate = `
resource "launchdarkly_custom_role" "test" {
	key = "%s"
	name = "Updated - %s"
	policy {
		actions = ["*"]	
		effect = "allow"
		resources = ["proj/*:env/staging"]
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
		resources = ["proj/*:env/staging"]
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
)

func TestAccCustomRole_Create(t *testing.T) {
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
				Config: fmt.Sprintf(testAccCustomRoleCreate, key, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, key),
					resource.TestCheckResourceAttr(resourceName, NAME, "Custom role - "+name),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "Deny all actions on production environments"),
					resource.TestCheckResourceAttr(resourceName, "policy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.resources.0", "proj/*:env/production"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.effect", "deny"),
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

func TestAccCustomRole_CreateWithNotStatements(t *testing.T) {
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
				Config: fmt.Sprintf(testAccCustomRoleCreateWithNotStatements, key, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, key),
					resource.TestCheckResourceAttr(resourceName, NAME, "Custom role - "+name),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "Don't allow all actions on non-staging environments"),
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
		},
	})
}

func TestAccCustomRole_Update(t *testing.T) {
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
				Config: fmt.Sprintf(testAccCustomRoleCreate, key, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(resourceName),
				),
			},
			{
				Config: fmt.Sprintf(testAccCustomRoleUpdate, key, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, key),
					resource.TestCheckResourceAttr(resourceName, NAME, "Updated - "+name),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, ""), // should be empty after removal
					resource.TestCheckResourceAttr(resourceName, "policy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.resources.0", "proj/*:env/staging"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.effect", "allow"),
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
				Config: fmt.Sprintf(testAccCustomRoleCreate, key, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(resourceName),
				),
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
		},
	})
}

func TestAccCustomRole_UpdateWithNotStatements(t *testing.T) {
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
				Config: fmt.Sprintf(testAccCustomRoleCreateWithStatements, key, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(resourceName),
				),
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
