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
	testAccDataSourceRelayProxyConfig = `
data "launchdarkly_relay_proxy_configuration" "test" {
	id = "%s"
}
`
)

func TestAccDataSourceRelayProxyConfig_noMatchReturnsError(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	invalidID := "31e801b0f65c6216806bd53b"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccDataSourceRelayProxyConfig, invalidID),
				ExpectError: regexp.MustCompile(fmt.Sprintf("Relay Proxy configuration with id %q not found", invalidID)),
			},
		},
	})
}

func TestAccDataSourceRelayProxyConfig_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false)
	require.NoError(t, err)

	name := "test config"
	resourceSpec := "proj/*:env/*"
	policy := []ldapi.StatementRep{{
		Resources: &([]string{resourceSpec}),
		Actions:   &([]string{"*"}),
		Effect:    "allow",
	}}

	post := ldapi.NewRelayAutoConfigPost(name, policy)
	config, _, err := client.ld.RelayProxyConfigurationsApi.PostRelayAutoConfig(client.ctx).RelayAutoConfigPost(*post).Execute()
	require.NoError(t, err)

	defer testAccDeleteRelayProxyConfig(t, client, config.Id)

	resourceName := "data.launchdarkly_relay_proxy_configuration.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceRelayProxyConfig, config.Id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, NAME, name),
					resource.TestCheckResourceAttr(resourceName, DISPLAY_KEY, config.DisplayKey),
					resource.TestCheckResourceAttr(resourceName, "policy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.effect", "allow"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.resources.0", resourceSpec),
					resource.TestCheckResourceAttr(resourceName, "policy.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.actions.0", "*"),
				),
			},
		},
	})
}

func TestAccDataSourceRelayProxyConfig_NotResource(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false)
	require.NoError(t, err)

	name := "test config"
	resourceSpec := "proj/*:env/*"
	policy := []ldapi.StatementRep{{
		NotResources: &([]string{resourceSpec}),
		Actions:      &([]string{"*"}),
		Effect:       "allow",
	}}

	post := ldapi.NewRelayAutoConfigPost(name, policy)
	config, _, err := client.ld.RelayProxyConfigurationsApi.PostRelayAutoConfig(client.ctx).RelayAutoConfigPost(*post).Execute()
	require.NoError(t, err)

	defer testAccDeleteRelayProxyConfig(t, client, config.Id)

	resourceName := "data.launchdarkly_relay_proxy_configuration.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceRelayProxyConfig, config.Id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, NAME, name),
					resource.TestCheckResourceAttr(resourceName, DISPLAY_KEY, config.DisplayKey),
					resource.TestCheckResourceAttr(resourceName, "policy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.effect", "allow"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.not_resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.not_resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.not_resources.0", resourceSpec),
					resource.TestCheckResourceAttr(resourceName, "policy.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.actions.0", "*"),
				),
			},
		},
	})
}

func testAccDeleteRelayProxyConfig(t *testing.T, client *Client, id string) {
	_, err := client.ld.RelayProxyConfigurationsApi.DeleteRelayAutoConfig(client.ctx, id).Execute()
	require.NoError(t, err)
}
