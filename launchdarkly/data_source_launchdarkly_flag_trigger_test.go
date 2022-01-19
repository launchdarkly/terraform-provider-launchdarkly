package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go/v7"
	"github.com/stretchr/testify/require"
)

const (
	testAccDataSourceFlagTrigger = `
data "launchdarkly_flag_trigger" "test" {
	project_key = "%s"
	env_key = "production"
	flag_key = "%s"
	id = "%s"
}	
`
)

func testAccDataSourceFlagTriggerScaffold(client *Client, projectKey, flagKey string, triggerBody *ldapi.TriggerPost) (*ldapi.TriggerWorkflowRep, error) {
	_, err := testAccDataSourceFeatureFlagScaffold(client, projectKey, *ldapi.NewFeatureFlagBody("Trigger Test", flagKey))
	if err != nil {
		return nil, err
	}
	trigger, _, err := client.ld.FlagTriggersApi.CreateTriggerWorkflow(client.ctx, projectKey, "production", flagKey).TriggerPost(*triggerBody).Execute()
	if err != nil {
		return nil, err
	}
	return &trigger, nil
}

func TestAccDataSourceFlagTrigger_noMatchReturnsError(t *testing.T) {
	id := "nonexistent-id"
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false)
	require.NoError(t, err)
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	flagKey := "trigger-test"
	_, err = testAccDataSourceFeatureFlagScaffold(client, projectKey, *ldapi.NewFeatureFlagBody("Trigger Test", flagKey))
	require.NoError(t, err)

	defer func() {
		err := testAccDataSourceProjectDelete(client, projectKey)
		require.NoError(t, err)
	}()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceFlagTrigger, projectKey, flagKey, id),
				// the integration key will not appear here since it is not set on the data source
				ExpectError: regexp.MustCompile(`Error: failed to get  trigger with ID `),
			},
		},
	})
}

func TestAccDataSourceFlagTrigger_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false)
	require.NoError(t, err)
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	flagKey := "trigger-test"
	instructions := []map[string]interface{}{{"kind": "turnFlagOff"}}
	post := ldapi.NewTriggerPost("datadog")
	post.Instructions = &instructions
	trigger, err := testAccDataSourceFlagTriggerScaffold(client, projectKey, flagKey, post)
	require.NoError(t, err)

	defer func() {
		err := testAccDataSourceProjectDelete(client, projectKey)
		require.NoError(t, err)
	}()

	resourceName := "data.launchdarkly_flag_trigger.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceFlagTrigger, projectKey, flagKey, *trigger.Id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", *trigger.Id),
					resource.TestCheckResourceAttrSet(resourceName, "maintainer_id"),
					resource.TestCheckResourceAttrSet(resourceName, "enabled"),
					resource.TestCheckResourceAttr(resourceName, "instructions.0.kind", "turnFlagOff"),
					resource.TestCheckResourceAttr(resourceName, "project_key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "env_key", "production"),
					resource.TestCheckResourceAttr(resourceName, "flag_key", flagKey),
					resource.TestCheckResourceAttr(resourceName, "integration_key", *trigger.IntegrationKey),
				),
			},
		},
	})

}
