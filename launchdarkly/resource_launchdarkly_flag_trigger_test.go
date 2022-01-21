package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	testAccFlagTriggerCreate = `
resource "launchdarkly_flag_trigger" "basic" {
	project_key = launchdarkly_project.test.key
	env_key = "test"
	flag_key = launchdarkly_feature_flag.trigger_flag.key
	integration_key = "generic-trigger"
	instructions {
		kind = "turnFlagOn"
	}
	enabled = false
}
`

	testAccFlagTriggerCreateEnabled = `
resource "launchdarkly_flag_trigger" "basic" {
	project_key = launchdarkly_project.test.key
	env_key = "test"
	flag_key = launchdarkly_feature_flag.trigger_flag.key
	integration_key = "generic-trigger"
	instructions {
		kind = "turnFlagOff"
	}
	enabled = true
}
`

	testAccFlagTriggerUpdate = `
resource "launchdarkly_flag_trigger" "basic" {
	project_key = launchdarkly_project.test.key
	env_key = "test"
	flag_key = launchdarkly_feature_flag.trigger_flag.key
	integration_key = "generic-trigger"
	instructions {
		kind = "turnFlagOff"
	}
	enabled = true
}
`

	testAccFlagTriggerUpdate2 = `
resource "launchdarkly_flag_trigger" "basic" {
	project_key = launchdarkly_project.test.key
	env_key = "test"
	flag_key = launchdarkly_feature_flag.trigger_flag.key
	integration_key = "generic-trigger"
	instructions {
		kind = "turnFlagOff"
	}
	enabled = false
}
`
)

func withRandomFlag(randomFlag, resource string) string {
	return fmt.Sprintf(`
		resource "launchdarkly_feature_flag" "trigger_flag" {
			project_key = launchdarkly_project.test.key
			key = "%s"
			name = "Basic feature flag"
			variation_type = "boolean"
		}
	
	%s`, randomFlag, resource)
}

func TestAccFlagTrigger_CreateUpdate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	flagKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_flag_trigger.basic"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, withRandomFlag(flagKey, testAccFlagTriggerCreate)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFlagExists(projectKey, "launchdarkly_feature_flag.trigger_flag"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, FLAG_KEY, flagKey),
					resource.TestCheckResourceAttr(resourceName, INTEGRATION_KEY, "generic-trigger"),
					resource.TestCheckResourceAttr(resourceName, "instructions.0.kind", "turnFlagOn"),
					resource.TestCheckResourceAttr(resourceName, ENABLED, "false"),
					resource.TestCheckResourceAttrSet(resourceName, TRIGGER_URL),
					resource.TestCheckResourceAttrSet(resourceName, MAINTAINER_ID),
				),
			},
			{
				Config: withRandomProject(projectKey, withRandomFlag(flagKey, testAccFlagTriggerUpdate)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFlagExists(projectKey, "launchdarkly_feature_flag.trigger_flag"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, FLAG_KEY, flagKey),
					resource.TestCheckResourceAttr(resourceName, INTEGRATION_KEY, "generic-trigger"),
					resource.TestCheckResourceAttr(resourceName, "instructions.0.kind", "turnFlagOff"),
					resource.TestCheckResourceAttr(resourceName, ENABLED, "true"),
					resource.TestCheckResourceAttrSet(resourceName, TRIGGER_URL),
					resource.TestCheckResourceAttrSet(resourceName, MAINTAINER_ID),
				),
			},
			{
				Config: withRandomProject(projectKey, withRandomFlag(flagKey, testAccFlagTriggerUpdate2)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFlagExists(projectKey, "launchdarkly_feature_flag.trigger_flag"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, FLAG_KEY, flagKey),
					resource.TestCheckResourceAttr(resourceName, INTEGRATION_KEY, "generic-trigger"),
					resource.TestCheckResourceAttr(resourceName, "instructions.0.kind", "turnFlagOff"),
					resource.TestCheckResourceAttr(resourceName, ENABLED, "false"),
					resource.TestCheckResourceAttrSet(resourceName, TRIGGER_URL),
					resource.TestCheckResourceAttrSet(resourceName, MAINTAINER_ID),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateIdPrefix:     fmt.Sprintf("%s/test/%s/", projectKey, flagKey),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{TRIGGER_URL},
			},
		},
	})
}

func TestAccFlagTrigger_CreateEnabled(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	flagKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_flag_trigger.basic"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, withRandomFlag(flagKey, testAccFlagTriggerCreateEnabled)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFlagExists(projectKey, "launchdarkly_feature_flag.trigger_flag"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, FLAG_KEY, flagKey),
					resource.TestCheckResourceAttr(resourceName, INTEGRATION_KEY, "generic-trigger"),
					resource.TestCheckResourceAttr(resourceName, "instructions.0.kind", "turnFlagOff"),
					resource.TestCheckResourceAttr(resourceName, ENABLED, "true"),
					resource.TestCheckResourceAttrSet(resourceName, TRIGGER_URL),
					resource.TestCheckResourceAttrSet(resourceName, MAINTAINER_ID),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateIdPrefix:     fmt.Sprintf("%s/test/%s/", projectKey, flagKey),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{TRIGGER_URL},
			},
		},
	})
}

func testAccCheckFlagExists(projectKey, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("flag ID is not set")
		}
		projectKey, flagKey, err := flagIdToKeys(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("flag ID is not set correctly")
		}

		client := testAccProvider.Meta().(*Client)
		_, _, err = client.ld.FeatureFlagsApi.GetFeatureFlag(client.ctx, projectKey, flagKey).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting flag. %s", err)
		}
		return nil
	}
}
