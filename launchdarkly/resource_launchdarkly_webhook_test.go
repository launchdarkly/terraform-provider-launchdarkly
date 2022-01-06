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
	on      = true
}	
`

	testAccWebhookCreateWithEnabled = `
resource "launchdarkly_webhook" "test" {
	name    = "example-webhook"
	url     = "http://webhooks.com"
	tags    = [ "terraform" ]
	on = true
}	
`

	testAccWebhookUpdate = `
resource "launchdarkly_webhook" "test" {
	name   = "Example Webhook"
	url    = "http://webhooks.com/updatedUrl"
	tags   = [ "terraform", "updated" ]
	on     = false
	secret = "SuperSecret"
}
`

	testAccWebhookWithStatements = `
resource "launchdarkly_webhook" "with_statements" {
	name    = "Webhook with policy statements"
	url     = "http://webhooks.com"
	on      = true
	statements {
		actions   = ["*"]	
		effect    = "allow"
		resources = ["proj/*:env/production:flag/*"]
	}
}
`

	testAccWebhookWithStatementsRemoved = `
resource "launchdarkly_webhook" "with_statements" {
	name    = "Webhook without statements"
	url     = "http://webhooks.com"
	on      = true
}
`

	testAccWebhookWithPolicyStatements = `
resource "launchdarkly_webhook" "with_statements" {
	name    = "Webhook with policy statements"
	url     = "http://webhooks.com"
	on      = true
	statements {
		actions   = ["*"]	
		effect    = "allow"
		resources = ["proj/*:env/production:flag/*"]
	}
}
`

	testAccWebhookWithPolicyUpdate = `
resource "launchdarkly_webhook" "with_statements" {
	name    = "Webhook with policy statements"
	url     = "http://webhooks.com"
	on      = true
	statements {
		actions   = ["*"]
		effect    = "allow"
		resources = ["proj/test:env/production:flag/*"]
	}
	statements {
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
	on      = true
	statements {
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
					resource.TestCheckResourceAttr(resourceName, NAME, "example-webhook"),
					resource.TestCheckResourceAttr(resourceName, URL, "http://webhooks.com"),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
					resource.TestCheckResourceAttr(resourceName, "statements.#", "0"),
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

func TestAccWebhook_CreateWithEnabled(t *testing.T) {
	resourceName := "launchdarkly_webhook.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccWebhookCreateWithEnabled,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWebhookExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "example-webhook"),
					resource.TestCheckResourceAttr(resourceName, URL, "http://webhooks.com"),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
					resource.TestCheckResourceAttr(resourceName, "statements.#", "0"),
				),
			},
			{
				ResourceName: resourceName,
				ImportState:  true,
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
				Config: testAccWebhookCreateWithEnabled,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWebhookExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "example-webhook"),
					resource.TestCheckResourceAttr(resourceName, URL, "http://webhooks.com"),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
					resource.TestCheckResourceAttr(resourceName, "statements.#", "0"),
				),
			},
			{
				Config: testAccWebhookCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWebhookExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "example-webhook"),
					resource.TestCheckResourceAttr(resourceName, URL, "http://webhooks.com"),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
					resource.TestCheckResourceAttr(resourceName, "statements.#", "0"),
				),
			},
			{
				Config: testAccWebhookUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWebhookExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Example Webhook"),
					resource.TestCheckResourceAttr(resourceName, URL, "http://webhooks.com/updatedUrl"),
					resource.TestCheckResourceAttr(resourceName, ON, "false"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "updated"),
					resource.TestCheckResourceAttr(resourceName, SECRET, "SuperSecret"),
					resource.TestCheckResourceAttr(resourceName, "statements.#", "0"),
				),
			},
			{
				ResourceName: resourceName,
				ImportState:  true,
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
				Config: testAccWebhookWithStatements,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWebhookExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Webhook with policy statements"),
					resource.TestCheckResourceAttr(resourceName, URL, "http://webhooks.com"),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "statements.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.effect", "allow"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.resources.0", "proj/*:env/production:flag/*"),
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

func TestAccWebhook_CreateWithPolicyStatements(t *testing.T) {
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
					resource.TestCheckResourceAttr(resourceName, NAME, "Webhook with policy statements"),
					resource.TestCheckResourceAttr(resourceName, URL, "http://webhooks.com"),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "statements.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.effect", "allow"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.resources.0", "proj/*:env/production:flag/*"),
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
				Config: testAccWebhookWithStatements,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWebhookExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Webhook with policy statements"),
					resource.TestCheckResourceAttr(resourceName, URL, "http://webhooks.com"),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "statements.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.effect", "allow"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.resources.0", "proj/*:env/production:flag/*"),
				),
			},
			{
				Config: testAccWebhookWithPolicyUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWebhookExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Webhook with policy statements"),
					resource.TestCheckResourceAttr(resourceName, URL, "http://webhooks.com"),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "statements.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.effect", "allow"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.resources.0", "proj/test:env/production:flag/*"),
					resource.TestCheckResourceAttr(resourceName, "statements.1.effect", "allow"),
					resource.TestCheckResourceAttr(resourceName, "statements.1.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "statements.1.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "statements.1.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "statements.1.resources.0", "proj/test:env/production:segment/*"),
				),
			},
			{
				Config: testAccWebhookWithStatementsRemoved,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWebhookExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Webhook without statements"),
					resource.TestCheckResourceAttr(resourceName, URL, "http://webhooks.com"),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "statements.#", "0"),
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

func TestAccWebhook_InvalidStatements(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccWebhookInvalidStatements,
				ExpectError: regexp.MustCompile("policy statements cannot contain both 'actions' and 'not_actions'"),
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
		_, _, err := client.ld.WebhooksApi.GetWebhook(client.ctx, rs.Primary.ID).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting webhook. %s", err)
		}
		return nil
	}
}
