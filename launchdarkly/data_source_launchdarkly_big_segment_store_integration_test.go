package launchdarkly

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// The data source looks up the integration created by the resource in the same
// config, threading the server-assigned integration_id through.
const testAccDataSourceBigSegmentStoreIntegration = `
resource "launchdarkly_big_segment_store_integration" "test" {
	project_key     = launchdarkly_project.test.key
	environment_key = "test"
	integration_key = "redis"
	name            = "Terraform Redis store"

	config = jsonencode({
		host = "redis.internal.example.com"
		port = 6379
		tls  = true
	})
}

data "launchdarkly_big_segment_store_integration" "test" {
	project_key     = launchdarkly_big_segment_store_integration.test.project_key
	environment_key = launchdarkly_big_segment_store_integration.test.environment_key
	integration_key = launchdarkly_big_segment_store_integration.test.integration_key
	integration_id  = launchdarkly_big_segment_store_integration.test.integration_id
}
`

func TestAccDataSourceBigSegmentStoreIntegration_basic(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	dataSourceName := "data.launchdarkly_big_segment_store_integration.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBigSegmentStoreIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccDataSourceBigSegmentStoreIntegration),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(dataSourceName, ENVIRONMENT_KEY, "test"),
					resource.TestCheckResourceAttr(dataSourceName, INTEGRATION_KEY, "redis"),
					resource.TestCheckResourceAttr(dataSourceName, NAME, "Terraform Redis store"),
					resource.TestCheckResourceAttrSet(dataSourceName, INTEGRATION_ID),
					resource.TestCheckResourceAttrSet(dataSourceName, VERSION),
					resource.TestCheckResourceAttrPair(dataSourceName, "id", "launchdarkly_big_segment_store_integration.test", "id"),
				),
			},
		},
	})
}
