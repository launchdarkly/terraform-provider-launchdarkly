package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// NOTE for reviewers (agent-scaffolded, verified against real LD): integration
// delivery configurations are scoped to a persistent feature store integration.
// The `integration_key` and the shape of `config` are defined by the
// integration's manifest. The feature store integrations exposed on the LD
// account are edge key-value providers (akamai-edgeworkers, cloudflare, convex,
// fastly, vercel, vercel-native) -- there is no `redis`/`dynamodb` feature store
// integration. We use `fastly` here because its manifest has no validation
// request, so a configuration can be created with placeholder credentials while
// `on = false` without LaunchDarkly attempting to reach the provider. Every
// feature store manifest declares at least one secret field (here `apiToken`),
// which the API returns obfuscated on read, so the secret-bearing `config`
// attribute is excluded from ImportStateVerify below.
const testAccIntegrationDeliveryConfigurationCreate = `
resource "launchdarkly_integration_delivery_configuration" "test" {
	project_key     = launchdarkly_project.test.key
	env_key         = "test"
	integration_key = "fastly"

	name = "Test Fastly feature store"
	on   = false

	config = jsonencode({
		storeId  = "00000000-0000-0000-0000-000000000000"
		apiToken = "dummy-token-for-acceptance-test"
	})

	tags = ["terraform-managed"]
}
`

const testAccIntegrationDeliveryConfigurationUpdate = `
resource "launchdarkly_integration_delivery_configuration" "test" {
	project_key     = launchdarkly_project.test.key
	env_key         = "test"
	integration_key = "fastly"

	name = "Test Fastly feature store updated"
	on   = false

	config = jsonencode({
		storeId  = "11111111-1111-1111-1111-111111111111"
		apiToken = "dummy-token-for-acceptance-test"
	})

	tags = ["terraform-managed", "updated"]
}
`

func TestAccIntegrationDeliveryConfiguration_CreateUpdate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_integration_delivery_configuration.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckIntegrationDeliveryConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccIntegrationDeliveryConfigurationCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckIntegrationDeliveryConfigurationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, INTEGRATION_KEY, "fastly"),
					resource.TestCheckResourceAttr(resourceName, NAME, "Test Fastly feature store"),
					resource.TestCheckResourceAttr(resourceName, ON, "false"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, CONFIG_ID),
					resource.TestCheckResourceAttrSet(resourceName, VERSION),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// The API obfuscates secret config fields (e.g. apiToken) on read,
				// so the imported `config` cannot round-trip to the original value.
				ImportStateVerifyIgnore: []string{CONFIG},
			},
			{
				Config: withRandomProject(projectKey, testAccIntegrationDeliveryConfigurationUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationDeliveryConfigurationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Test Fastly feature store updated"),
					resource.TestCheckResourceAttr(resourceName, ON, "false"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{CONFIG},
			},
		},
	})
}

func testAccCheckIntegrationDeliveryConfigurationExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		projectKey, envKey, integrationKey, configID, err := integrationDeliveryConfigurationIDToKeys(rs.Primary.ID)
		if err != nil {
			return err
		}
		beta, err := newIntegrationDeliveryConfigurationBetaClient(mustTestAccClient())
		if err != nil {
			return err
		}
		_, _, err = beta.ld.IntegrationDeliveryConfigurationsBetaApi.
			GetIntegrationDeliveryConfigurationById(beta.ctx, projectKey, envKey, integrationKey, configID).
			Execute()
		if err != nil {
			return fmt.Errorf("received an error getting integration delivery configuration: %s", err)
		}
		return nil
	}
}

func testAccCheckIntegrationDeliveryConfigurationDestroy(s *terraform.State) error {
	beta, err := newIntegrationDeliveryConfigurationBetaClient(mustTestAccClient())
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_integration_delivery_configuration" {
			continue
		}
		projectKey, envKey, integrationKey, configID, err := integrationDeliveryConfigurationIDToKeys(rs.Primary.ID)
		if err != nil {
			return err
		}
		_, res, err := beta.ld.IntegrationDeliveryConfigurationsBetaApi.
			GetIntegrationDeliveryConfigurationById(beta.ctx, projectKey, envKey, integrationKey, configID).
			Execute()
		if isStatusNotFound(res) {
			continue
		}
		if err != nil {
			return fmt.Errorf("unexpected error checking integration delivery configuration %q destruction: %s", configID, handleLdapiErr(err))
		}
		return fmt.Errorf("integration delivery configuration %q still exists", configID)
	}
	return nil
}
