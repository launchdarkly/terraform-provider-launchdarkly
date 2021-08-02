package launchdarkly

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const (
	testAccAccessTokenCreate = `
resource "launchdarkly_access_token" "test" {
	name = "Access token - %s"
	role = "reader"
}
`
	testAccAccessTokenCreateWithImmutableParams = `
resource "launchdarkly_access_token" "test" {
	name = "Access token - %s"
	role = "reader"
	service_token = true
	default_api_version = 20160426
}
`
	testAccAccessTokenCreateWithCustomRole = `
resource "launchdarkly_custom_role" "role" {
	key = "%s"
	name = "Custom role - %s"
	description = "Deny all actions on production environments"
	policy_statements {
		actions = ["*"]
		effect = "deny"
		resources = ["proj/*:env/production"]
	}
}

resource "launchdarkly_access_token" "test" {
	name = "Access token - %s"
	custom_roles = [launchdarkly_custom_role.role.key]
}
`
	testAccAccessTokenUpdateCustomRole = `
resource "launchdarkly_custom_role" "role" {
	key = "%s"
	name = "Custom role - %s"
	description= "Deny all actions on production environments"
	policy_statements {
		actions = ["*"]
		effect = "deny"
		resources = ["proj/*:env/production"]
	}
}

resource "launchdarkly_custom_role" "role2" {
	key = "%s2"
	name = "Custom role - %s2"
	description= "Deny all actions on production environments"
	policy_statements {
		actions = ["*"]
		effect = "deny"
		resources = ["proj/*:env/production"]
	}
}

resource "launchdarkly_access_token" "test" {
	name = "Updated - %s"
	custom_roles = [launchdarkly_custom_role.role.key, launchdarkly_custom_role.role2.key]
}
`
	testAccAccessTokenUpdate = `
resource "launchdarkly_access_token" "test" {
	name = "Updated - %s"
	inline_roles {
		actions = ["*"]
		effect = "deny"
		resources = ["proj/*:env/production"]
	}
}
`
	testAccAccessTokenCreateWithInlineRoles = `
resource "launchdarkly_access_token" "test" {
	name = "Access token - %s"
	inline_roles {
		actions = ["*"]
		effect = "allow"
		resources = ["proj/*:env/staging"]
	}
}
`
	testAccAccessTokenCreateWithPolicyStatements = `
resource "launchdarkly_access_token" "test" {
	name = "Access token - %s"
	policy_statements {
		actions = ["*"]
		effect = "allow"
		resources = ["proj/*:env/staging"]
	}
}
`
	testAccAccessTokenReset = `
resource "launchdarkly_access_token" "test" {
	name = "Access token - %s"
	role = "reader"
	expire = %d
}
`
)

func TestAccAccessToken_Create(t *testing.T) {
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_access_token.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAccessTokenCreate, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccessTokenExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "Access token - "+name),
					resource.TestCheckResourceAttr(resourceName, "role", "reader"),
					resource.TestCheckResourceAttr(resourceName, "service_token", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "default_api_version"),
					resource.TestCheckResourceAttrSet(resourceName, "token"),
					resource.TestCheckNoResourceAttr(resourceName, "policy"),
					resource.TestCheckNoResourceAttr(resourceName, "custom_roles"),
				),
			},
		},
	})
}

func TestAccAccessToken_CreateWithCustomRole(t *testing.T) {
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_access_token.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAccessTokenCreateWithCustomRole, name, name, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccessTokenExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "Access token - "+name),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "service_token", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "default_api_version"),
					resource.TestCheckResourceAttrSet(resourceName, "token"),
					resource.TestCheckNoResourceAttr(resourceName, "policy"),
					resource.TestCheckNoResourceAttr(resourceName, "role"),
				),
			},
		},
	})
}

func TestAccAccessToken_CreateWithImmutableParams(t *testing.T) {
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_access_token.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAccessTokenCreateWithImmutableParams, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccessTokenExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "Access token - "+name),
					resource.TestCheckResourceAttr(resourceName, "role", "reader"),
					resource.TestCheckResourceAttr(resourceName, "service_token", "true"),
					resource.TestCheckResourceAttr(resourceName, "default_api_version", "20160426"),
					resource.TestCheckResourceAttrSet(resourceName, "token"),
					resource.TestCheckNoResourceAttr(resourceName, "policy"),
					resource.TestCheckNoResourceAttr(resourceName, "custom_roles"),
				),
			},
		},
	})
}

func TestAccAccessToken_CreateWithInlineRoles(t *testing.T) {
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_access_token.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAccessTokenCreateWithInlineRoles, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccessTokenExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "Access token - "+name),
					resource.TestCheckResourceAttr(resourceName, "inline_roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "inline_roles.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "inline_roles.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "inline_roles.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "inline_roles.0.resources.0", "proj/*:env/staging"),
					resource.TestCheckResourceAttr(resourceName, "inline_roles.0.effect", "allow"),
					resource.TestCheckResourceAttr(resourceName, "service_token", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "default_api_version"),
					resource.TestCheckResourceAttrSet(resourceName, "token"),
					resource.TestCheckNoResourceAttr(resourceName, "role"),
					resource.TestCheckNoResourceAttr(resourceName, "custom_roles"),
				),
			},
		},
	})
}

func TestAccAccessToken_CreateWithPolicyStatements(t *testing.T) {
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_access_token.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAccessTokenCreateWithPolicyStatements, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccessTokenExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "Access token - "+name),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.resources.0", "proj/*:env/staging"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.effect", "allow"),
					resource.TestCheckResourceAttr(resourceName, "service_token", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "default_api_version"),
					resource.TestCheckResourceAttrSet(resourceName, "token"),
					resource.TestCheckNoResourceAttr(resourceName, "role"),
					resource.TestCheckNoResourceAttr(resourceName, "custom_roles"),
				),
			},
		},
	})
}

func TestAccAccessToken_Update(t *testing.T) {
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_access_token.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAccessTokenCreate, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccessTokenExists(resourceName),
				),
			},
			{
				Config: fmt.Sprintf(testAccAccessTokenUpdate, name), // update regular role to policy_statements roles
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccessTokenExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "Updated - "+name),
					resource.TestCheckResourceAttr(resourceName, "inline_roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "inline_roles.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "inline_roles.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "inline_roles.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "inline_roles.0.resources.0", "proj/*:env/production"),
					resource.TestCheckResourceAttr(resourceName, "inline_roles.0.effect", "deny"),
				),
			},
		},
	})
}

func TestAccAccessToken_UpdateCustomRole(t *testing.T) {
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_access_token.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAccessTokenCreateWithCustomRole, name, name, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccessTokenExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_roles."+testAccMemberCustomRolePropertyKey(name), name),
				),
			},
			{
				Config: fmt.Sprintf(testAccAccessTokenUpdateCustomRole, name, name, name, name, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccessTokenExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "Updated - "+name),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "custom_roles."+testAccMemberCustomRolePropertyKey(name), name),
					resource.TestCheckResourceAttr(resourceName, "custom_roles."+testAccMemberCustomRolePropertyKey(name+"2"), name+"2"),
				),
			},
		},
	})
}

func TestAccAccessToken_Reset(t *testing.T) {
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_access_token.test"

	original := new(string)
	updated := new(string)

	hourFromNow := time.Now().Add(time.Hour).Unix() * 1000
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAccessTokenCreate, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccessTokenExists(resourceName),
					testAccStoreAccessTokenSecret(original, resourceName),
				),
			},
			{
				Config: fmt.Sprintf(testAccAccessTokenReset, name, -1),
				Check: resource.ComposeTestCheckFunc(
					testAccStoreAccessTokenSecret(updated, resourceName),
					testAccCheckAccessTokenChanged(original, updated),
					resource.TestCheckResourceAttr(resourceName, "expire", "-1"),
					// reset the original secret for the next test
					testAccStoreAccessTokenSecret(original, resourceName),
				),
			},
			{
				Config: fmt.Sprintf(testAccAccessTokenReset, name, hourFromNow),
				Check: resource.ComposeTestCheckFunc(
					testAccStoreAccessTokenSecret(updated, resourceName),
					testAccCheckAccessTokenChanged(original, updated),
					resource.TestCheckResourceAttr(resourceName, "expire", strconv.Itoa(int(hourFromNow))),
				),
			},
		},
	})
}

func testAccStoreAccessTokenSecret(ptr *string, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		*ptr = rs.Primary.Attributes[TOKEN]
		return nil
	}
}

func testAccCheckAccessTokenChanged(original, updated *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *original == *updated {
			return fmt.Errorf("access token secret did not changed")
		}
		return nil
	}
}

func testAccCheckAccessTokenExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("access token ID is not set")
		}
		client := testAccProvider.Meta().(*Client)
		_, _, err := client.ld.AccessTokensApi.GetToken(client.ctx, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("received an error getting access token. %s", err)
		}
		return nil
	}
}
