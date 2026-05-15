package launchdarkly

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	testAccCustomRoleCreate = `
	resource "launchdarkly_custom_role" "test" {
		key = "%s"
		name = "Custom role - %s"
		description = "Deny all actions on production environments"
		base_permissions = "no_access"
		policy = [{
			actions = ["*"]	
			effect = "deny"
			resources = ["proj/*:env/production"]
		}]
	}
`
	// IMPORTANT TO NOTE that the $ character must be escaped in terraform by using a double $$
	// otherwas ${} will be interpreted as a terraform variable and throw an error
	testAccCustomRoleUpdate = `
resource "launchdarkly_custom_role" "test" {
	key = "%s"
	name = "Updated - %s"
	policy = [{
		actions = ["*"]	
		effect = "allow"
		resources = ["proj/*:env/$${roleAttribute/devEnvironments}"]
	}]
}
`
	testAccCustomRoleCreateWithStatements = `
resource "launchdarkly_custom_role" "test" {
	key = "%s"
	name = "Custom role - %s"
	description = "Allow all actions on staging environments"
	policy_statements = [{
		actions = ["*"]	
		effect = "allow"
		resources = ["proj/$${roleAttribute/devProjects}:env/staging"]
	}]
}
`
	testAccCustomRoleUpdateWithStatements = `
resource "launchdarkly_custom_role" "test" {
	key = "%s"
	name = "Updated role - %s"
	description= "Deny all actions on production environments"
	policy_statements = [{
		actions = ["*"]	
		effect = "deny"
		resources = ["proj/*:env/production"]
	}]
}
`
	testAccCustomRoleCreateWithNotStatements = `
resource "launchdarkly_custom_role" "test" {
	key = "%s"
	name = "Custom role - %s"
	description = "Don't allow all actions on non-staging environments"
	policy_statements = [{
		not_actions = ["*"]	
		effect = "allow"
		not_resources = ["proj/*:env/staging"]
	}]
}
`
	testAccCustomRoleUpdateWithNotStatements = `
resource "launchdarkly_custom_role" "test" {
	key = "%s"
	name = "Updated role - %s"
	description= "Don't deny all actions on non production environments"
	policy_statements = [{
		not_actions = ["*"]
		effect = "deny"
		not_resources = ["proj/*:env/production"]
	}]
}
`
	testAccCustomRoleCreateWithStatementsJSON = `
resource "launchdarkly_custom_role" "test" {
	key         = "%s"
	name        = "JSON role - %s"
	description = "Allow actions on staging via JSON policy"
	policy_statements_json = jsonencode([
		{
			effect    = "allow",
			resources = ["proj/$${roleAttribute/devProjects}:env/staging"],
			actions   = ["*"]
		}
	])
}
`
	testAccCustomRoleUpdateWithStatementsJSON = `
resource "launchdarkly_custom_role" "test" {
	key         = "%s"
	name        = "Updated JSON role - %s"
	description = "Deny actions on production via JSON policy"
	policy_statements_json = jsonencode([
		{
			effect    = "deny",
			resources = ["proj/*:env/production"],
			actions   = ["*"]
		},
		{
			effect       = "allow",
			not_resources = ["proj/*:env/production"],
			actions       = ["updateOn"]
		}
	])
}
`
	testAccCustomRoleConflictingForms = `
resource "launchdarkly_custom_role" "test" {
	key  = "%s"
	name = "Conflict - %s"
	policy_statements = [{
		actions   = ["*"]
		effect    = "allow"
		resources = ["proj/*"]
	}]
	policy_statements_json = jsonencode([
		{
			effect    = "allow",
			resources = ["proj/*"],
			actions   = ["*"]
		}
	])
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
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCustomRoleDestroy,
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
					// Removed from config; framework writes null (absent) on Read
					// when the API returns empty description. SDKv2 used to write
					// "" here, but null is the framework-correct state for an
					// Optional-only attribute. See framework_helpers.go for
					// stringValueOrNullFromPointer.
					resource.TestCheckNoResourceAttr(resourceName, DESCRIPTION),
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
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCustomRoleDestroy,
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
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCustomRoleDestroy,
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

func TestAccCustomRole_JSONPolicy(t *testing.T) {
	key := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_custom_role.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCustomRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCustomRoleCreateWithStatementsJSON, key, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, key),
					resource.TestCheckResourceAttr(resourceName, NAME, "JSON role - "+name),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "Allow actions on staging via JSON policy"),
					resource.TestCheckResourceAttr(resourceName, "policy.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.#", "0"),
					resource.TestCheckResourceAttrSet(resourceName, POLICY_STATEMENTS_JSON),
					testAccCheckCustomRolePolicyAPI(resourceName, []map[string]interface{}{
						{
							"effect":    "allow",
							"resources": []interface{}{"proj/${roleAttribute/devProjects}:env/staging"},
							"actions":   []interface{}{"*"},
						},
					}),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{POLICY_STATEMENTS_JSON, POLICY_STATEMENTS},
			},
			{
				Config: fmt.Sprintf(testAccCustomRoleUpdateWithStatementsJSON, key, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Updated JSON role - "+name),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "Deny actions on production via JSON policy"),
					resource.TestCheckResourceAttr(resourceName, "policy.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.#", "0"),
					resource.TestCheckResourceAttrSet(resourceName, POLICY_STATEMENTS_JSON),
					testAccCheckCustomRolePolicyAPI(resourceName, []map[string]interface{}{
						{
							"effect":    "deny",
							"resources": []interface{}{"proj/*:env/production"},
							"actions":   []interface{}{"*"},
						},
						{
							"effect":        "allow",
							"not_resources": []interface{}{"proj/*:env/production"},
							"actions":       []interface{}{"updateOn"},
						},
					}),
				),
			},
		},
	})
}

func TestAccCustomRole_JSONPolicyConflictsWithPolicyStatements(t *testing.T) {
	key := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccCustomRoleConflictingForms, key, name),
				ExpectError: regexp.MustCompile(`(?s)Conflicting policy fields`),
			},
		},
	})
}

// testAccCheckCustomRolePolicyAPI fetches the role from the LD API and asserts
// the returned Policy matches the expected statement set (order-independent).
func testAccCheckCustomRolePolicyAPI(resourceName string, expected []map[string]interface{}) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		client := mustTestAccClient()
		role, _, err := client.ld.CustomRolesApi.GetCustomRole(client.ctx, rs.Primary.ID).Execute()
		if err != nil {
			return fmt.Errorf("could not fetch custom role %q: %s", rs.Primary.ID, err)
		}
		if len(role.Policy) != len(expected) {
			return fmt.Errorf("policy length mismatch: got %d, want %d (API: %+v)", len(role.Policy), len(expected), role.Policy)
		}
		actualNorms := make([]string, len(role.Policy))
		for i, st := range role.Policy {
			m := map[string]interface{}{"effect": st.Effect}
			if len(st.Resources) > 0 {
				m["resources"] = toIfaceSlice(st.Resources)
			}
			if len(st.NotResources) > 0 {
				m["not_resources"] = toIfaceSlice(st.NotResources)
			}
			if len(st.Actions) > 0 {
				m["actions"] = toIfaceSlice(st.Actions)
			}
			if len(st.NotActions) > 0 {
				m["not_actions"] = toIfaceSlice(st.NotActions)
			}
			b, _ := json.Marshal(m)
			actualNorms[i] = string(b)
		}
		for _, e := range expected {
			b, _ := json.Marshal(e)
			want := string(b)
			found := false
			for _, got := range actualNorms {
				if got == want {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("expected statement not found in API response: want=%s actual=%v", want, actualNorms)
			}
		}
		return nil
	}
}

func toIfaceSlice(in []string) []interface{} {
	out := make([]interface{}, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
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
		client := mustTestAccClient()
		_, _, err := client.ld.CustomRolesApi.GetCustomRole(client.ctx, rs.Primary.ID).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting custom role. %s", err)
		}
		return nil
	}
}

// testAccCheckCustomRoleDestroy verifies the custom role has been destroyed
func testAccCheckCustomRoleDestroy(s *terraform.State) error {
	client := mustTestAccClient()
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
