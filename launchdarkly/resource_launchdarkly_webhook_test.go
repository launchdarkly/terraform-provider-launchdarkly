package launchdarkly

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	testAccWebhookCreate = `
resource "launchdarkly_webhook" "test" {
	name    = "example-webhook"
	url     = "http://webhooks.com"
	tags    = [ "terraform" ]
	enabled = true
}
`

	testAccWebhookUpdate = `
resource "launchdarkly_webhook" "test" {
	name = "Example Webhook"
	url = "http://webhooks.com/updatedUrl"
	tags = [ "terraform", "updated" ]
	enabled = false
	secret = "SuperSecret"
}
`

	testAccWebhookWithPolicyStatements = `
resource "launchdarkly_webhook" "with_statements" {
	name    = "Webhook with policy statements"
	url     = "http://webhooks.com"
	enabled = true
	policy_statements {
		actions   = ["*"]
		effect    = "allow"
		resources = ["proj/*:env/production:flag/*"]
	}
}
`

	testAccWebhookWithPolicyStatementsUpdate = `
resource "launchdarkly_webhook" "with_statements" {
	name    = "Webhook with policy statements"
	url     = "http://webhooks.com"
	enabled = true
	policy_statements {
		actions   = ["*"]
		effect    = "allow"
		resources = ["proj/test:env/production:flag/*"]
	}
	policy_statements {
		actions   = ["*"]
		effect    = "allow"
		resources = ["proj/test:env/production:segment/*"]
	}
}
`

	testAccWebhookInvalidStatements = `
resource "launchdarkly_webhook" "with_statements" {
	name    = "Webhook with policy statements"
	url     = "http://webhooks.com"
	enabled = true
	policy_statements {
		actions   = ["*"]
		not_actions = ["*"]
		effect    = "allow"
		resources = ["proj/*:env/production:flag/*"]
	}
}
`
)

func TestAccWebhook_Create(t *testing.T) {
	resourceName := "launchdarkly_webhook.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccWebhookCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWebhookExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example-webhook"),
					resource.TestCheckResourceAttr(resourceName, "url", "http://webhooks.com"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("terraform"), "terraform"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.#", "0"),
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

func TestAccWebhook_Update(t *testing.T) {
	resourceName := "launchdarkly_webhook.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccWebhookCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWebhookExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example-webhook"),
					resource.TestCheckResourceAttr(resourceName, "url", "http://webhooks.com"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("terraform"), "terraform"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.#", "0"),
				),
			},
			{
				Config: testAccWebhookUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWebhookExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "Example Webhook"),
					resource.TestCheckResourceAttr(resourceName, "url", "http://webhooks.com/updatedUrl"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("terraform"), "terraform"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("updated"), "updated"),
					resource.TestCheckResourceAttr(resourceName, SECRET, "SuperSecret"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.#", "0"),
				),
			},
		},
	})
}

func TestAccWebhook_CreateWithStatements(t *testing.T) {
	resourceName := "launchdarkly_webhook.with_statements"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccWebhookWithPolicyStatements,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWebhookExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "Webhook with policy statements"),
					resource.TestCheckResourceAttr(resourceName, "url", "http://webhooks.com"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.effect", "allow"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.resources.0", "proj/*:env/production:flag/*"),
				),
			},
		},
	})
}

func TestAccWebhook_UpdateWithStatements(t *testing.T) {
	resourceName := "launchdarkly_webhook.with_statements"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccWebhookWithPolicyStatements,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWebhookExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "Webhook with policy statements"),
					resource.TestCheckResourceAttr(resourceName, "url", "http://webhooks.com"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.effect", "allow"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.resources.0", "proj/*:env/production:flag/*"),
				),
			},
			{
				Config: testAccWebhookWithPolicyStatementsUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWebhookExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "Webhook with policy statements"),
					resource.TestCheckResourceAttr(resourceName, "url", "http://webhooks.com"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.effect", "allow"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.resources.0", "proj/test:env/production:flag/*"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.1.effect", "allow"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.1.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.1.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.1.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.1.resources.0", "proj/test:env/production:segment/*"),
				),
			},
		},
	})
}

func TestAccWebhook_InvalidStatements(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccWebhookInvalidStatements,
				ExpectError: regexp.MustCompile("policy_statements cannot contain both 'actions' and 'not_actions'"),
			},
		},
	})
}

func testAccCheckWebhookExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("webhook ID is not set")
		}
		client := testAccProvider.Meta().(*Client)
		_, _, err := client.ld.WebhooksApi.GetWebhook(client.ctx, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("received an error getting webhook. %s", err)
		}
		return nil
	}
}
