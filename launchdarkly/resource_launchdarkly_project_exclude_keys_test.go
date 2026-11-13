package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// testAccProjectExcludeKeys renders a project with two environments.
// "secret-env" carries the given exclude_keys_from_state line ("" omits it);
// "plain-env" never sets it, so it must keep storing its keys throughout.
func testAccProjectExcludeKeys(projectKey, excludeLine string) string {
	return fmt.Sprintf(`
resource "launchdarkly_project" "exclude_keys" {
	key  = "%s"
	name = "exclude keys project"
	environments = {
		"secret-env" = {
			name  = "Secret Environment"
			color = "010101"
			%s
		}
		"plain-env" = {
			name  = "Plain Environment"
			color = "020202"
		}
	}
}
`, projectKey, excludeLine)
}

func TestAccProject_ExcludeKeysFromState(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_project.exclude_keys"
	secretAPIKey := "environments.secret-env." + API_KEY
	secretMobileKey := "environments.secret-env." + MOBILE_KEY
	secretExclude := "environments.secret-env." + EXCLUDE_KEYS_FROM_STATE
	plainAPIKey := "environments.plain-env." + API_KEY
	plainMobileKey := "environments.plain-env." + MOBILE_KEY
	var apiKey, mobileKey, plainKey string

	secretEnvPath := tfjsonpath.New(ENVIRONMENTS).AtMapKey("secret-env")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckProjectDestroy,
		Steps: []resource.TestStep{
			// Unset: historical behavior, every environment stores its keys.
			{
				Config: testAccProjectExcludeKeys(projectKey, ""),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckNoResourceAttr(resourceName, secretExclude),
					resource.TestCheckResourceAttrSet(resourceName, secretAPIKey),
					resource.TestCheckResourceAttrSet(resourceName, secretMobileKey),
					resource.TestCheckResourceAttrSet(resourceName, plainAPIKey),
					testAccCaptureAttr(resourceName, secretAPIKey, &apiKey),
					testAccCaptureAttr(resourceName, secretMobileKey, &mobileKey),
					testAccCaptureAttr(resourceName, plainAPIKey, &plainKey),
				),
			},
			// Opt one environment in, in place: only its secret keys leave state.
			{
				Config: testAccProjectExcludeKeys(projectKey, "exclude_keys_from_state = true"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
						plancheck.ExpectKnownValue(resourceName, secretEnvPath.AtMapKey(API_KEY), knownvalue.Null()),
						plancheck.ExpectKnownValue(resourceName, secretEnvPath.AtMapKey(MOBILE_KEY), knownvalue.Null()),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, secretExclude, "true"),
					resource.TestCheckNoResourceAttr(resourceName, secretAPIKey),
					resource.TestCheckNoResourceAttr(resourceName, secretMobileKey),
					resource.TestCheckResourceAttrSet(resourceName, "environments.secret-env."+CLIENT_SIDE_ID),
					testAccCheckAttrEqualsCaptured(resourceName, plainAPIKey, &plainKey),
					resource.TestCheckResourceAttrSet(resourceName, plainMobileKey),
					testAccCheckEnvironmentKeysUnchangedInLD(projectKey, "secret-env", &apiKey, &mobileKey),
				),
			},
			// Import cannot see the config, so every environment's keys are
			// stored and exclude_keys_from_state is null.
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					secretAPIKey, secretMobileKey, secretExclude,
				},
			},
			// Opt back out: keys return to state, unchanged in LaunchDarkly.
			{
				Config: testAccProjectExcludeKeys(projectKey, "exclude_keys_from_state = false"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
						plancheck.ExpectUnknownValue(resourceName, secretEnvPath.AtMapKey(API_KEY)),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, secretExclude, "false"),
					testAccCheckAttrEqualsCaptured(resourceName, secretAPIKey, &apiKey),
					testAccCheckAttrEqualsCaptured(resourceName, secretMobileKey, &mobileKey),
					testAccCheckAttrEqualsCaptured(resourceName, plainAPIKey, &plainKey),
					testAccCheckEnvironmentKeysUnchangedInLD(projectKey, "secret-env", &apiKey, &mobileKey),
				),
			},
			// And back in again.
			{
				Config: testAccProjectExcludeKeys(projectKey, "exclude_keys_from_state = true"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(resourceName, secretAPIKey),
					resource.TestCheckNoResourceAttr(resourceName, secretMobileKey),
					testAccCheckAttrEqualsCaptured(resourceName, plainAPIKey, &plainKey),
				),
			},
		},
	})
}

func TestAccProject_ExcludeKeysFromStateOnCreate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_project.exclude_keys"
	secretEnvPath := tfjsonpath.New(ENVIRONMENTS).AtMapKey("secret-env")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectExcludeKeys(projectKey, "exclude_keys_from_state = true"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionCreate),
						plancheck.ExpectKnownValue(resourceName, secretEnvPath.AtMapKey(API_KEY), knownvalue.Null()),
						plancheck.ExpectKnownValue(resourceName, secretEnvPath.AtMapKey(MOBILE_KEY), knownvalue.Null()),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "environments.secret-env."+EXCLUDE_KEYS_FROM_STATE, "true"),
					resource.TestCheckNoResourceAttr(resourceName, "environments.secret-env."+API_KEY),
					resource.TestCheckNoResourceAttr(resourceName, "environments.secret-env."+MOBILE_KEY),
					resource.TestCheckResourceAttrSet(resourceName, "environments.secret-env."+CLIENT_SIDE_ID),
					resource.TestCheckResourceAttrSet(resourceName, "environments.plain-env."+API_KEY),
					resource.TestCheckResourceAttrSet(resourceName, "environments.plain-env."+MOBILE_KEY),
				),
			},
		},
	})
}
