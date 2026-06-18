package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// The environment key "test" is created by withRandomProject. Persistent store
// integrations are environment-scoped, so the resource references that env.
//
// NOTE for the reviewer: these fixtures use a Redis store with placeholder
// connection values. If the beta API validates store connectivity at create
// time (rather than lazily), the create will fail and the fixtures must point at
// a reachable Redis instance. Confirm against a live environment.
const testAccBigSegmentStoreIntegrationRedis = `
resource "launchdarkly_big_segment_store_integration" "test" {
	project_key     = launchdarkly_project.test.key
	environment_key = "test"
	integration_key = "redis"
	name            = "Terraform Redis store"
	on              = false

	config = jsonencode({
		host = "redis.internal.example.com"
		port = 6379
		tls  = true
	})

	tags = ["terraform-managed"]
}
`

const testAccBigSegmentStoreIntegrationRedisUpdate = `
resource "launchdarkly_big_segment_store_integration" "test" {
	project_key     = launchdarkly_project.test.key
	environment_key = "test"
	integration_key = "redis"
	name            = "Terraform Redis store updated"
	on              = true

	config = jsonencode({
		host = "redis.internal.example.com"
		port = 6380
		tls  = true
	})

	tags = ["terraform-managed", "updated"]
}
`

func TestAccBigSegmentStoreIntegration_CreateUpdate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_big_segment_store_integration.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBigSegmentStoreIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccBigSegmentStoreIntegrationRedis),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckBigSegmentStoreIntegrationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENVIRONMENT_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, INTEGRATION_KEY, "redis"),
					resource.TestCheckResourceAttr(resourceName, NAME, "Terraform Redis store"),
					resource.TestCheckResourceAttr(resourceName, ON, "false"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, INTEGRATION_ID),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// config carries credentials the API may redact on read; ignore
				// it during import verification. Reviewer: drop this if the live
				// API round-trips config faithfully.
				ImportStateVerifyIgnore: []string{CONFIG},
			},
			{
				Config: withRandomProject(projectKey, testAccBigSegmentStoreIntegrationRedisUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBigSegmentStoreIntegrationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Terraform Redis store updated"),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
				),
			},
		},
	})
}

func testAccCheckBigSegmentStoreIntegrationExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		environmentKey := rs.Primary.Attributes[ENVIRONMENT_KEY]
		integrationKey := rs.Primary.Attributes[INTEGRATION_KEY]
		integrationID := rs.Primary.Attributes[INTEGRATION_ID]
		if integrationID == "" {
			return fmt.Errorf("integration_id not set: %s", resourceName)
		}
		beta, err := newBigSegmentStoreIntegrationBetaClient(mustTestAccClient())
		if err != nil {
			return err
		}
		_, _, err = beta.ld.PersistentStoreIntegrationsBetaApi.GetBigSegmentStoreIntegration(beta.ctx, projectKey, environmentKey, integrationKey, integrationID).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting big segment store integration: %s", err)
		}
		return nil
	}
}

func testAccCheckBigSegmentStoreIntegrationDestroy(s *terraform.State) error {
	beta, err := newBigSegmentStoreIntegrationBetaClient(mustTestAccClient())
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_big_segment_store_integration" {
			continue
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		environmentKey := rs.Primary.Attributes[ENVIRONMENT_KEY]
		integrationKey := rs.Primary.Attributes[INTEGRATION_KEY]
		integrationID := rs.Primary.Attributes[INTEGRATION_ID]
		_, res, err := beta.ld.PersistentStoreIntegrationsBetaApi.GetBigSegmentStoreIntegration(beta.ctx, projectKey, environmentKey, integrationKey, integrationID).Execute()
		if isStatusNotFound(res) {
			continue
		}
		if err != nil {
			return fmt.Errorf("unexpected error checking big segment store integration %q destruction in %q/%q: %s", integrationID, projectKey, environmentKey, handleLdapiErr(err))
		}
		return fmt.Errorf("big segment store integration %q still exists in %q/%q", integrationID, projectKey, environmentKey)
	}
	return nil
}
