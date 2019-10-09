package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const (
	testAccWebhookCreate = `
resource "launchdarkly_webhook" "test" {
	name = "example-webhook"
	url = "http://webhooks.com"
	tags = [ "terraform" ]
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
}`
)

func TestAccWebhook_Create(t *testing.T) {
	resourceName := "launchdarkly_webhook.test"
	resource.Test(t, resource.TestCase{
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
				),
			},
		},
	})
}

func TestAccWebhook_Update(t *testing.T) {
	resourceName := "launchdarkly_webhook.test"
	resource.Test(t, resource.TestCase{
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
					resource.TestCheckResourceAttr(resourceName, secret, "SuperSecret"),
				),
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
