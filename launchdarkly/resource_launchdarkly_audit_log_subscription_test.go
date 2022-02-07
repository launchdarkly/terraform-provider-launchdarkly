package launchdarkly

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	testAccAuditLogSubscriptionCreate = `
resource "launchdarkly_audit_log_subscription" "%s_tf_test" {
	integration_key = "%s"
	name = "terraform test"
	config = %s
	tags = [
		"integrations",
		"terraform"
	]
	on = true
	statements {
		actions = ["*"]
		effect = "deny"
		resources = ["proj/*:env/*:flag/*"]
	}
}
`

	testAccAuditLogSubscriptionUpdate = `
resource "launchdarkly_audit_log_subscription" "%s_tf_test" {
	integration_key = "%s"
	name = "terraform test v2"
	config = %s
	on = false
	tags = [
		"integrations"
	]
	statements {
		actions = ["*"]
		effect = "allow"
		resources = ["proj/*:env/production"]
	}
}
`
)

func TestAccAuditLogSubscription_CreateUpdateDatadog(t *testing.T) {
	integrationKey := "datadog"
	// omitting host_url = "https://api.datadoghq.com" to test the handling of attributes with default values
	config := `{
		api_key = "thisisasecretkey"
	}		
	`

	resourceName := fmt.Sprintf("launchdarkly_audit_log_subscription.%s_tf_test", integrationKey)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAuditLogSubscriptionCreate, integrationKey, integrationKey, config),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, ID),
					resource.TestCheckResourceAttr(resourceName, INTEGRATION_KEY, integrationKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "terraform test"),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "config.api_key", "thisisasecretkey"),
					// resource.TestCheckResourceAttr(resourceName, "config.host_url", "https://api.datadoghq.com"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "integrations"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "terraform"),
					resource.TestCheckResourceAttr(resourceName, "statements.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.resources.0", "proj/*:env/*:flag/*"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.effect", "deny"),
				),
			},
			{
				Config: fmt.Sprintf(testAccAuditLogSubscriptionUpdate, integrationKey, integrationKey, config),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, ID),
					resource.TestCheckResourceAttr(resourceName, INTEGRATION_KEY, integrationKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "terraform test v2"),
					resource.TestCheckResourceAttr(resourceName, ON, "false"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "integrations"),
					resource.TestCheckResourceAttr(resourceName, "statements.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.effect", "allow"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.resources.0", "proj/*:env/production"),
				),
			},
		},
	})
}

func TestAccAuditLogSubscription_CreateDynatrace(t *testing.T) {
	integrationKey := "dynatrace"
	config := `{
		api_token = "verysecrettoken"
		url = "https://launchdarkly.appdynamics.com"
		entity = "APPLICATION_METHOD"
	}		
	`

	resourceName := fmt.Sprintf("launchdarkly_audit_log_subscription.%s_tf_test", integrationKey)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAuditLogSubscriptionCreate, integrationKey, integrationKey, config),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, ID),
					resource.TestCheckResourceAttr(resourceName, INTEGRATION_KEY, integrationKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "terraform test"),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "config.api_token", "verysecrettoken"),
					resource.TestCheckResourceAttr(resourceName, "config.url", "https://launchdarkly.appdynamics.com"),
					resource.TestCheckResourceAttr(resourceName, "config.entity", "APPLICATION_METHOD"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "integrations"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "terraform"),
					resource.TestCheckResourceAttr(resourceName, "statements.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.resources.0", "proj/*:env/*:flag/*"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.effect", "deny"),
				),
			},
		},
	})
}

func TestAccAuditLogSubscription_CreateMSTeams(t *testing.T) {
	integrationKey := "msteams"
	config := `{
		url = "https://outlook.office.com/webhook/terraform-test"
	}		
	`

	resourceName := fmt.Sprintf("launchdarkly_audit_log_subscription.%s_tf_test", integrationKey)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAuditLogSubscriptionCreate, integrationKey, integrationKey, config),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, ID),
					resource.TestCheckResourceAttr(resourceName, INTEGRATION_KEY, integrationKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "terraform test"),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "config.url", "https://outlook.office.com/webhook/terraform-test"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "integrations"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "terraform"),
					resource.TestCheckResourceAttr(resourceName, "statements.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.resources.0", "proj/*:env/*:flag/*"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.effect", "deny"),
				),
			},
		},
	})
}

func TestAccAuditLogSubscription_CreateSlack(t *testing.T) {
	// splunk specifically needs to be converted to kebab case, so we need to handle it specially
	integrationKey := "slack"
	config := `{
		url = "https://hooks.slack.com/services/SOME-RANDOM-HOOK"
	}		
	`

	resourceName := fmt.Sprintf("launchdarkly_audit_log_subscription.%s_tf_test", integrationKey)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAuditLogSubscriptionCreate, integrationKey, integrationKey, config),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, ID),
					resource.TestCheckResourceAttr(resourceName, INTEGRATION_KEY, integrationKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "terraform test"),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "config.url", "https://hooks.slack.com/services/SOME-RANDOM-HOOK"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "integrations"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "terraform"),
					resource.TestCheckResourceAttr(resourceName, "statements.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.resources.0", "proj/*:env/*:flag/*"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.effect", "deny"),
				),
			},
		},
	})
}

func TestAccAuditLogSubscription_CreateSplunk(t *testing.T) {
	// splunk specifically needs to be converted to kebab case, so we need to handle it specially
	integrationKey := "splunk"
	config := `{
		base_url = "https://launchdarkly.splunk.com"
		token = "averysecrettoken"
		skip_ca_verification = true
	}		
	`

	resourceName := fmt.Sprintf("launchdarkly_audit_log_subscription.%s_tf_test", integrationKey)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccAuditLogSubscriptionCreate, integrationKey, integrationKey, config),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, ID),
					resource.TestCheckResourceAttr(resourceName, INTEGRATION_KEY, integrationKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "terraform test"),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "config.base_url", "https://launchdarkly.splunk.com"),
					resource.TestCheckResourceAttr(resourceName, "config.token", "averysecrettoken"),
					resource.TestCheckResourceAttr(resourceName, "config.skip_ca_verification", "true"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "integrations"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "terraform"),
					resource.TestCheckResourceAttr(resourceName, "statements.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.actions.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.resources.0", "proj/*:env/*:flag/*"),
					resource.TestCheckResourceAttr(resourceName, "statements.0.effect", "deny"),
				),
			},
		},
	})
}

func TestAccAuditLogSubscription_WrongConfigReturnsError(t *testing.T) {
	integrationKey := "honeycomb"
	config := `{
		url = "https://bad-config.com/terraform-test"
	}		
	`
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccAuditLogSubscriptionCreate, integrationKey, integrationKey, config),
				ExpectError: regexp.MustCompile(`Error: failed to create honeycomb integration with name terraform test: config variable url not valid for integration type honeycomb`),
			},
		},
	})
}

func testAccCheckIntegrationExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		integrationKey, ok := rs.Primary.Attributes[INTEGRATION_KEY]
		if !ok {
			return fmt.Errorf("integration integrationKey not found: %s", resourceName)
		}
		integrationID, ok := rs.Primary.Attributes[ID]
		if !ok {
			return fmt.Errorf("integration not found: %s", resourceName)
		}
		client := testAccProvider.Meta().(*Client)
		_, _, err := client.ld.IntegrationAuditLogSubscriptionsApi.GetSubscriptionByID(client.ctx, integrationKey, integrationID).Execute()
		if err != nil {
			return fmt.Errorf("error getting %s integration: %s", integrationKey, err)
		}

		return nil
	}
}
