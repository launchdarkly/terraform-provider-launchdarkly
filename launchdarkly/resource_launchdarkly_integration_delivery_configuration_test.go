package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// NOTE for reviewers (agent-scaffolded): integration delivery configurations are
// scoped to a persistent feature store integration. The `integration_key` and
// the shape of `config` are defined by the integration's manifest, and the
// account running these acceptance tests must have the `redis` feature store
// integration available. If the dedicated test account exposes a different
// feature store integration, swap the integration key and config below to match
// that integration's manifest form variables.
const testAccIntegrationDeliveryConfigurationCreate = `
resource "launchdarkly_integration_delivery_configuration" "test" {
	project_key     = launchdarkly_project.test.key
	env_key         = "test"
	integration_key = "redis"

	name = "Test Redis feature store"
	on   = false

	config = jsonencode({
		host   = "redis.example.com"
		port   = 6379
		prefix = "launchdarkly"
	})

	tags = ["terraform-managed"]
}
`

const testAccIntegrationDeliveryConfigurationUpdate = `
resource "launchdarkly_integration_delivery_configuration" "test" {
	project_key     = launchdarkly_project.test.key
	env_key         = "test"
	integration_key = "redis"

	name = "Test Redis feature store updated"
	on   = true

	config = jsonencode({
		host   = "redis-updated.example.com"
		port   = 6380
		prefix = "launchdarkly-prod"
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
					resource.TestCheckResourceAttr(resourceName, INTEGRATION_KEY, "redis"),
					resource.TestCheckResourceAttr(resourceName, NAME, "Test Redis feature store"),
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
			},
			{
				Config: withRandomProject(projectKey, testAccIntegrationDeliveryConfigurationUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationDeliveryConfigurationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Test Redis feature store updated"),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
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
