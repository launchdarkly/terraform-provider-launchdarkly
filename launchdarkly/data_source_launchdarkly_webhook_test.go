package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go/v14"
	"github.com/stretchr/testify/require"
)

const (
	testAccDataSourceWebhook = `
data "launchdarkly_webhook" "test" {
	id = "%s"
}	
`
)

func testAccDataSourceWebhookCreate(client *Client, webhookName string) (*ldapi.Webhook, error) {
	statementResources := []string{"proj/*"}
	statementActions := []string{"updateOn"}
	webhookBody := ldapi.WebhookPost{
		Url:  "https://www.example.com",
		Sign: false,
		On:   true,
		Name: ldapi.PtrString(webhookName),
		Tags: []string{"terraform"},
		Statements: []ldapi.StatementPost{
			{
				Resources: statementResources,
				Actions:   statementActions,
				Effect:    "allow",
			},
		},
	}
	webhook, _, err := client.ld.WebhooksApi.PostWebhook(client.ctx).WebhookPost(webhookBody).Execute()

	if err != nil {
		return nil, fmt.Errorf("failed to create webhook with name %q: %s", webhookName, handleLdapiErr(err))
	}

	return webhook, nil
}

func testAccDataSourceWebhookDelete(client *Client, webhookId string) error {
	_, err := client.ld.WebhooksApi.DeleteWebhook(client.ctx, webhookId).Execute()

	if err != nil {
		return fmt.Errorf("failed to delete webhook with id %q: %s", webhookId, handleLdapiErr(err))
	}
	return nil
}

func TestAccDataSourceWebhook_noMatchReturnsError(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	webhookId := acctest.RandStringFromCharSet(24, acctest.CharSetAlphaNum)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccDataSourceWebhook, webhookId),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`Error: failed to get webhook with id "%s": 404 Not Found:`, webhookId)),
			},
		},
	})
}

func TestAccDataSourceWebhook_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	webhookName := "Data Source Test"
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S)
	require.NoError(t, err)
	webhook, err := testAccDataSourceWebhookCreate(client, webhookName)
	require.NoError(t, err)
	defer func() {
		err := testAccDataSourceWebhookDelete(client, webhook.Id)
		require.NoError(t, err)
	}()

	resourceName := "data.launchdarkly_webhook.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceWebhook, webhook.Id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, ID),
					resource.TestCheckResourceAttr(resourceName, ID, webhook.Id),
					resource.TestCheckResourceAttr(resourceName, NAME, webhookName),
					resource.TestCheckResourceAttr(resourceName, URL, webhook.Url),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "statements.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.resources.0", "proj/*"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.actions.0", "updateOn"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.effect", "allow"),
					resource.TestCheckResourceAttr(resourceName, SECRET, ""), // since we set Sign to false

				),
			},
		},
	})
}
