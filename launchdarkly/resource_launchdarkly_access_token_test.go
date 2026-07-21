package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
	default_api_version = 20240415
}
`
	testAccAccessTokenCreateWithCustomRole = `
resource "launchdarkly_custom_role" "role" {
	key = "%s"
	name = "Custom role - %s"
	description = "Deny all actions on production environments"
	policy_statements = [{
		actions = ["*"]
		effect = "deny"
		resources = ["proj/*:env/production"]
	}]
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
	policy_statements = [{
		actions = ["*"]
		effect = "deny"
		resources = ["proj/*:env/production"]
	}]
}

resource "launchdarkly_custom_role" "role2" {
	key = "%s2"
	name = "Custom role - %s2"
	description= "Deny all actions on production environments"
	policy_statements = [{
		actions = ["*"]
		effect = "deny"
		resources = ["proj/*:env/production"]
	}]
}

resource "launchdarkly_access_token" "test" {
	name = "Updated - %s"
	custom_roles = [launchdarkly_custom_role.role.key, launchdarkly_custom_role.role2.key]
}
`
	testAccAccessTokenUpdate = `
resource "launchdarkly_access_token" "test" {
	name = "Updated - %s"
	inline_roles = [{
		actions = ["*"]
		effect = "deny"
		resources = ["proj/*:env/production"]
	}]
}
`
	testAccAccessTokenCreateWithInlineRoles = `
resource "launchdarkly_access_token" "test" {
	name = "Access token - %s"
	inline_roles = [{
		actions = ["*"]
		effect = "allow"
		resources = ["proj/*:env/staging"]
	}]
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
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAccessTokenDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAccessTokenCreate, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccessTokenExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Access token - "+name),
					resource.TestCheckResourceAttr(resourceName, ROLE, "reader"),
					resource.TestCheckResourceAttr(resourceName, SERVICE_TOKEN, "false"),
					resource.TestCheckResourceAttrSet(resourceName, DEFAULT_API_VERSION),
					resource.TestCheckResourceAttrSet(resourceName, TOKEN),
					resource.TestCheckNoResourceAttr(resourceName, POLICY),
					resource.TestCheckNoResourceAttr(resourceName, CUSTOM_ROLES),
				),
			},
		},
	})
}

func TestAccAccessToken_WithCustomRole(t *testing.T) {
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_access_token.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAccessTokenDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAccessTokenCreateWithCustomRole, name, name, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccessTokenExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Access token - "+name),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, SERVICE_TOKEN, "false"),
					resource.TestCheckResourceAttrSet(resourceName, DEFAULT_API_VERSION),
					resource.TestCheckResourceAttrSet(resourceName, TOKEN),
					resource.TestCheckNoResourceAttr(resourceName, POLICY),
					resource.TestCheckNoResourceAttr(resourceName, ROLE),
				),
			},
			{
				Config: fmt.Sprintf(testAccAccessTokenUpdateCustomRole, name, name, name, name, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccessTokenExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Updated - "+name),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.0", name),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.1", name+"2"),
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
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAccessTokenDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAccessTokenCreateWithImmutableParams, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccessTokenExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Access token - "+name),
					resource.TestCheckResourceAttr(resourceName, ROLE, "reader"),
					resource.TestCheckResourceAttr(resourceName, SERVICE_TOKEN, "true"),
					resource.TestCheckResourceAttr(resourceName, DEFAULT_API_VERSION, "20240415"),
					resource.TestCheckResourceAttrSet(resourceName, TOKEN),
					resource.TestCheckNoResourceAttr(resourceName, POLICY),
					resource.TestCheckNoResourceAttr(resourceName, CUSTOM_ROLES),
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
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAccessTokenDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAccessTokenCreateWithInlineRoles, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccessTokenExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Access token - "+name),
					resource.TestCheckResourceAttr(resourceName, "inline_roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "inline_roles.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "inline_roles.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "inline_roles.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "inline_roles.0.resources.0", "proj/*:env/staging"),
					resource.TestCheckResourceAttr(resourceName, "inline_roles.0.effect", "allow"),
					resource.TestCheckResourceAttr(resourceName, SERVICE_TOKEN, "false"),
					resource.TestCheckResourceAttrSet(resourceName, DEFAULT_API_VERSION),
					resource.TestCheckResourceAttrSet(resourceName, TOKEN),
					resource.TestCheckNoResourceAttr(resourceName, ROLE),
					resource.TestCheckNoResourceAttr(resourceName, CUSTOM_ROLES),
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
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAccessTokenDestroy,
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
					resource.TestCheckResourceAttr(resourceName, NAME, "Updated - "+name),
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

func testAccCheckAccessTokenExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("access token ID is not set")
		}
		client := mustTestAccClient()
		_, _, err := client.ld.AccessTokensApi.GetToken(client.ctx, rs.Primary.ID).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting access token. %s", err)
		}
		return nil
	}
}

// testAccCheckAccessTokenDestroy verifies the access token has been destroyed
func testAccCheckAccessTokenDestroy(s *terraform.State) error {
	client := mustTestAccClient()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_access_token" {
			continue
		}

		_, res, err := client.ld.AccessTokensApi.GetToken(client.ctx, rs.Primary.ID).Execute()

		if isStatusNotFound(res) {
			continue
		}

		if err != nil {
			return err
		}

		return fmt.Errorf("access token %s still exists", rs.Primary.ID)
	}
	return nil
}
