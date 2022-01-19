package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go/v7"
	"github.com/stretchr/testify/require"
)

const (
	testAccDataSourceAuditLogSubscriptionBasic = `
data "launchdarkly_audit_log_subscription" "test" {
	id = "%s"
	integration_key = "%s"
}
`

	testAccDataSourceAuditLogSubscriptionExists = `
data "launchdarkly_audit_log_subscription" "test" {
		id = "%s"
		integration_key = "%s"
	}
	`
)

func testAccDataSourceAuditLogSubscriptionCreate(client *Client, integrationKey string, subscriptionBody ldapi.SubscriptionPost) (*ldapi.Integration, error) {
	statementResources := []string{"proj/*"}
	statementActions := []string{"*"}
	statements := []ldapi.StatementPost{{
		Effect:    "allow",
		Resources: &statementResources,
		Actions:   &statementActions,
	}}
	subscriptionBody.Statements = &statements

	sub, _, err := client.ld.IntegrationAuditLogSubscriptionsApi.CreateSubscription(client.ctx, integrationKey).SubscriptionPost(subscriptionBody).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to create integration subscription for test: %v", handleLdapiErr(err))
	}
	return &sub, nil
}

func testAccDataSourceAuditLogSubscriptionDelete(client *Client, integrationKey, id string) error {
	_, err := client.ld.IntegrationAuditLogSubscriptionsApi.DeleteSubscription(client.ctx, integrationKey, id).Execute()

	if err != nil {
		return fmt.Errorf("failed to delete integration with ID %q: %s", id, handleLdapiErr(err))
	}
	return nil
}

func TestAccDataSourceAuditLogSubscription_noMatchReturnsError(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}
	id := "fake-id"
	integrationKey := "msteams"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccDataSourceAuditLogSubscriptionBasic, id, integrationKey),
				ExpectError: regexp.MustCompile(`Error: failed to get integration with ID "fake-id": 404 Not Found`),
			},
		},
	})
}

func TestAccDataSourceAuditLogSubscription_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	integrationKey := "datadog"
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false)
	require.NoError(t, err)

	subscriptionBody := ldapi.SubscriptionPost{
		Name: "test subscription",
		Config: map[string]interface{}{
			"apiKey":  "thisisasecretkey",
			"hostURL": "https://api.datadoghq.com",
		},
	}
	sub, err := testAccDataSourceAuditLogSubscriptionCreate(client, integrationKey, subscriptionBody)
	require.NoError(t, err)

	defer func() {
		err := testAccDataSourceAuditLogSubscriptionDelete(client, integrationKey, *sub.Id)
		require.NoError(t, err)
	}()

	resourceName := "data.launchdarkly_audit_log_subscription.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceAuditLogSubscriptionExists, *sub.Id, integrationKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "id", *sub.Id),
				),
			},
		},
	})
}
