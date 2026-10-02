package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// testAccEnvironmentExcludeKeys renders a launchdarkly_environment with the
// given exclude_keys_from_state line ("" omits the attribute entirely).
func testAccEnvironmentExcludeKeys(excludeLine string) string {
	return fmt.Sprintf(`
resource "launchdarkly_environment" "exclude_keys" {
	name        = "Exclude Keys"
	key         = "exclude-keys"
	color       = "ababab"
	project_key = launchdarkly_project.test.key
	%s
}
`, excludeLine)
}

// testAccCaptureAttr copies a state attribute into dst so later steps can
// compare against it.
func testAccCaptureAttr(resourceName, attr string, dst *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		v, ok := rs.Primary.Attributes[attr]
		if !ok || v == "" {
			return fmt.Errorf("%s: attribute %q is not set", resourceName, attr)
		}
		*dst = v
		return nil
	}
}

// testAccCheckAttrEqualsCaptured compares a state attribute against a value
// captured by testAccCaptureAttr in an earlier step. The pointer is read when
// the check runs, not when the test case is built.
func testAccCheckAttrEqualsCaptured(resourceName, attr string, want *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		return resource.TestCheckResourceAttr(resourceName, attr, *want)(s)
	}
}

// testAccCheckEnvironmentKeysUnchangedInLD asserts the environment's keys in
// LaunchDarkly still match the values captured earlier, proving that toggling
// exclude_keys_from_state never resets or rotates keys.
func testAccCheckEnvironmentKeysUnchangedInLD(projectKey, envKey string, apiKey, mobileKey *string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client := mustTestAccClient()
		env, _, err := client.ld.EnvironmentsApi.GetEnvironment(client.ctx, projectKey, envKey).Execute()
		if err != nil {
			return fmt.Errorf("failed to get environment %q in project %q: %s", envKey, projectKey, handleLdapiErr(err))
		}
		if env.ApiKey != *apiKey {
			return fmt.Errorf("environment SDK key changed in LaunchDarkly")
		}
		if env.MobileKey != *mobileKey {
			return fmt.Errorf("environment mobile key changed in LaunchDarkly")
		}
		return nil
	}
}

func TestAccEnvironment_ExcludeKeysFromStateToggle(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_environment.exclude_keys"
	envKey := "exclude-keys"
	var apiKey, mobileKey string

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Unset: historical behavior, keys are stored in state.
			{
				Config: withRandomProject(projectKey, testAccEnvironmentExcludeKeys("")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckNoResourceAttr(resourceName, EXCLUDE_KEYS_FROM_STATE),
					resource.TestCheckResourceAttrSet(resourceName, API_KEY),
					resource.TestCheckResourceAttrSet(resourceName, MOBILE_KEY),
					resource.TestCheckResourceAttrSet(resourceName, CLIENT_SIDE_ID),
					testAccCaptureAttr(resourceName, API_KEY, &apiKey),
					testAccCaptureAttr(resourceName, MOBILE_KEY, &mobileKey),
				),
			},
			// Explicit false: identical to unset.
			{
				Config: withRandomProject(projectKey, testAccEnvironmentExcludeKeys("exclude_keys_from_state = false")),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, EXCLUDE_KEYS_FROM_STATE, "false"),
					testAccCheckAttrEqualsCaptured(resourceName, API_KEY, &apiKey),
					testAccCheckAttrEqualsCaptured(resourceName, MOBILE_KEY, &mobileKey),
				),
			},
			// Opt in: keys are removed from state in place, without replacement.
			{
				Config: withRandomProject(projectKey, testAccEnvironmentExcludeKeys("exclude_keys_from_state = true")),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
						plancheck.ExpectKnownValue(resourceName, tfjsonpath.New(API_KEY), knownvalue.Null()),
						plancheck.ExpectKnownValue(resourceName, tfjsonpath.New(MOBILE_KEY), knownvalue.Null()),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, EXCLUDE_KEYS_FROM_STATE, "true"),
					resource.TestCheckNoResourceAttr(resourceName, API_KEY),
					resource.TestCheckNoResourceAttr(resourceName, MOBILE_KEY),
					resource.TestCheckResourceAttrSet(resourceName, CLIENT_SIDE_ID),
					testAccCheckEnvironmentKeysUnchangedInLD(projectKey, envKey, &apiKey, &mobileKey),
				),
			},
			// Import cannot see the config, so the imported state stores the
			// keys and leaves exclude_keys_from_state null.
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: append([]string{
					API_KEY, MOBILE_KEY, EXCLUDE_KEYS_FROM_STATE,
				}, importIgnoreApprovalSettingsKeys...),
			},
			// Opt back out: keys return to state, unchanged in LaunchDarkly.
			{
				Config: withRandomProject(projectKey, testAccEnvironmentExcludeKeys("exclude_keys_from_state = false")),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, EXCLUDE_KEYS_FROM_STATE, "false"),
					testAccCheckAttrEqualsCaptured(resourceName, API_KEY, &apiKey),
					testAccCheckAttrEqualsCaptured(resourceName, MOBILE_KEY, &mobileKey),
					testAccCheckEnvironmentKeysUnchangedInLD(projectKey, envKey, &apiKey, &mobileKey),
				),
			},
			// Removing the attribute again is a no-op for the keys.
			{
				Config: withRandomProject(projectKey, testAccEnvironmentExcludeKeys("")),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(resourceName, EXCLUDE_KEYS_FROM_STATE),
					testAccCheckAttrEqualsCaptured(resourceName, API_KEY, &apiKey),
					testAccCheckAttrEqualsCaptured(resourceName, MOBILE_KEY, &mobileKey),
				),
			},
		},
	})
}

func TestAccEnvironment_ExcludeKeysFromStateOnCreate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_environment.exclude_keys"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccEnvironmentExcludeKeys("exclude_keys_from_state = true")),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionCreate),
						plancheck.ExpectKnownValue(resourceName, tfjsonpath.New(API_KEY), knownvalue.Null()),
						plancheck.ExpectKnownValue(resourceName, tfjsonpath.New(MOBILE_KEY), knownvalue.Null()),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, EXCLUDE_KEYS_FROM_STATE, "true"),
					resource.TestCheckNoResourceAttr(resourceName, API_KEY),
					resource.TestCheckNoResourceAttr(resourceName, MOBILE_KEY),
					resource.TestCheckResourceAttrSet(resourceName, CLIENT_SIDE_ID),
				),
			},
			// Updating an unrelated attribute keeps the keys out of state.
			{
				Config: withRandomProject(projectKey, `
resource "launchdarkly_environment" "exclude_keys" {
	name                    = "Exclude Keys Renamed"
	key                     = "exclude-keys"
	color                   = "ababab"
	project_key             = launchdarkly_project.test.key
	exclude_keys_from_state = true
}
`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
						plancheck.ExpectKnownValue(resourceName, tfjsonpath.New(API_KEY), knownvalue.Null()),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, NAME, "Exclude Keys Renamed"),
					resource.TestCheckNoResourceAttr(resourceName, API_KEY),
					resource.TestCheckNoResourceAttr(resourceName, MOBILE_KEY),
				),
			},
		},
	})
}
