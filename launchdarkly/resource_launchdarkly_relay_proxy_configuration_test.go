package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	testAccRelayProxyConfigCreate = `
resource "launchdarkly_relay_proxy_configuration" "test" {
	name = "example-config"
	policy {
		actions   = ["*"]	
		effect    = "allow"
		resources = ["proj/*:env/*"]
	}
}
`

	testAccRelayProxyConfigUpdate = `
resource "launchdarkly_relay_proxy_configuration" "test" {
	name = "updated-config"
	policy {
		not_actions   = ["*"]	
		effect        = "deny"
		not_resources = ["proj/*:env/test"]
	}
}
`
)

func getRelayProxyConfigImportStep(resourceName string) resource.TestStep {
	return resource.TestStep{
		ResourceName:      resourceName,
		ImportState:       true,
		ImportStateVerify: true,
		// Because the FULL_KEY is only revealed when the config is created we will never be able to import it
		ImportStateVerifyIgnore: []string{FULL_KEY},
	}
}

func TestAccRelayProxyConfig_Create(t *testing.T) {
	resourceName := "launchdarkly_relay_proxy_configuration.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccRelayProxyConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRelayProxyConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example-config"),
					resource.TestCheckResourceAttrSet(resourceName, "full_key"),
					resource.TestCheckResourceAttrSet(resourceName, "display_key"),
					resource.TestCheckResourceAttr(resourceName, "policy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.effect", "allow"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.resources.0", "proj/*:env/*"),
				),
			},
			getRelayProxyConfigImportStep(resourceName),
		},
	},
	)
}

func TestAccRelayProxyConfig_Update(t *testing.T) {
	resourceName := "launchdarkly_relay_proxy_configuration.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccRelayProxyConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRelayProxyConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example-config"),
					resource.TestCheckResourceAttrSet(resourceName, "full_key"),
					resource.TestCheckResourceAttrSet(resourceName, "display_key"),
					resource.TestCheckResourceAttr(resourceName, "policy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.effect", "allow"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.resources.0", "proj/*:env/*"),
				),
			},
			getRelayProxyConfigImportStep(resourceName),
			{
				Config: testAccRelayProxyConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRelayProxyConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "updated-config"),
					resource.TestCheckResourceAttrSet(resourceName, "full_key"),
					resource.TestCheckResourceAttrSet(resourceName, "display_key"),
					resource.TestCheckResourceAttr(resourceName, "policy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.actions.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.not_actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.not_actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.effect", "deny"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.not_resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.not_resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy.0.not_resources.0", "proj/*:env/test"),
				),
			},
			getRelayProxyConfigImportStep(resourceName),
		},
	},
	)
}

func testAccCheckRelayProxyConfigExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("webhook ID is not set")
		}

		client := testAccProvider.Meta().(*Client)
		_, _, err := client.ld.RelayProxyConfigurationsApi.GetRelayProxyConfig(client.ctx, rs.Primary.ID).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting relay proxy config: %w", err)
		}

		return nil
	}
}
