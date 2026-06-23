package launchdarkly

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go/v22"
	"github.com/stretchr/testify/require"
)

const testAccDataSourceIntegrationDeliveryConfiguration = `
data "launchdarkly_integration_delivery_configuration" "testing" {
	project_key     = "%s"
	env_key         = "%s"
	integration_key = "%s"
	config_id       = "%s"
}
`

// testAccDataSourceIntegrationDeliveryConfigurationScaffold creates a project and
// a delivery configuration via the API so the data source can read it back. The
// delivery configuration endpoints are beta, so the config is created with the
// beta client.
//
// NOTE for reviewers: uses `fastly` to match the resource acceptance test. The
// dedicated LD account exposes edge key-value feature store integrations
// (akamai-edgeworkers, cloudflare, convex, fastly, vercel, vercel-native) and
// has no `redis` integration; `fastly` has no validation request so a config
// can be created with placeholder credentials while `on = false`.
func testAccDataSourceIntegrationDeliveryConfigurationScaffold(client *Client, beta *Client, projectKey, envKey, integrationKey string) (*ldapi.IntegrationDeliveryConfiguration, error) {
	projectBody := ldapi.ProjectPost{
		Name: "Integration Delivery Configuration Test Project",
		Key:  projectKey,
	}
	if _, err := testAccProjectScaffoldCreate(client, projectBody); err != nil {
		return nil, err
	}

	post := ldapi.NewIntegrationDeliveryConfigurationPost(map[string]interface{}{
		"storeId":  "00000000-0000-0000-0000-000000000000",
		"apiToken": "dummy-token-for-acceptance-test",
	})
	post.Name = ldapi.PtrString("DS Test Fastly feature store")
	post.On = ldapi.PtrBool(false)
	post.Tags = []string{"test"}

	cfg, _, err := beta.ld.IntegrationDeliveryConfigurationsBetaApi.
		CreateIntegrationDeliveryConfiguration(beta.ctx, projectKey, envKey, integrationKey).
		IntegrationDeliveryConfigurationPost(*post).
		Execute()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func TestAccDataSourceIntegrationDeliveryConfiguration_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	envKey := "test"
	integrationKey := "fastly"
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)
	beta, err := newIntegrationDeliveryConfigurationBetaClient(client)
	require.NoError(t, err)

	cfg, err := testAccDataSourceIntegrationDeliveryConfigurationScaffold(client, beta, projectKey, envKey, integrationKey)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, testAccProjectScaffoldDelete(client, projectKey))
	}()

	resourceName := "data.launchdarkly_integration_delivery_configuration.testing"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceIntegrationDeliveryConfiguration, projectKey, envKey, integrationKey, cfg.GetId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, envKey),
					resource.TestCheckResourceAttr(resourceName, INTEGRATION_KEY, integrationKey),
					resource.TestCheckResourceAttr(resourceName, CONFIG_ID, cfg.GetId()),
					resource.TestCheckResourceAttr(resourceName, NAME, cfg.GetName()),
					resource.TestCheckResourceAttr(resourceName, ID, integrationDeliveryConfigurationID(projectKey, envKey, integrationKey, cfg.GetId())),
					resource.TestCheckResourceAttrSet(resourceName, VERSION),
				),
			},
		},
	})
}
