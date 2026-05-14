package launchdarkly

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	testAccFeatureFlagDeprecated = `
resource "launchdarkly_feature_flag" "deprecated" {
	project_key = launchdarkly_project.test.key
	key = "deprecated-flag"
	name = "Deprecated feature flag"
	variation_type = "boolean"
	variations = [
		{ value = "true" },
		{ value = "false" },
	]
	deprecated = true
}
`

	testAccFeatureFlagUndeprecated = `
resource "launchdarkly_feature_flag" "deprecated" {
	project_key = launchdarkly_project.test.key
	key = "deprecated-flag"
	name = "Deprecated feature flag"
	variation_type = "boolean"
	variations = [
		{ value = "true" },
		{ value = "false" },
	]
	deprecated = false
}
`

	testAccFeatureFlagBasic = `
resource "launchdarkly_feature_flag" "basic" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag"
	name = "Basic feature flag"
	variation_type = "boolean"
	variations = [
		{ value = "true" },
		{ value = "false" },
	]
}
`
	testAccFeatureFlagBasicWithTag = `
resource "launchdarkly_feature_flag" "basic" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag"
	name = "Basic feature flag"
	variation_type = "boolean"
	variations = [
		{ value = "true" },
		{ value = "false" },
	]
	tags = ["test"]
}
`

	testAccFeatureFlagBasicWithCSASet = `
resource "launchdarkly_feature_flag" "basic" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag"
	name = "Basic feature flag"
	variation_type = "boolean"
	variations = [
		{ value = "true" },
		{ value = "false" },
	]
	tags = ["test"]

	client_side_availability = [{
		using_environment_id = true
		using_mobile_key = true
	}]
}
`

	testAccFeatureFlagUpdate = `
resource "launchdarkly_feature_flag" "basic" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag"
	name = "Less basic feature flag"
	variation_type = "boolean"
	variations = [
		{ value = "true" },
		{ value = "false" },
	]
	description = "this is a boolean flag by default because the variations field is omitted"
	tags = ["update", "terraform"]
	include_in_snippet = true
	temporary = true
	defaults = [{
		on_variation = 1
		off_variation = 1
	}]
}
`

	testAccFeatureFlagNumber = `
resource "launchdarkly_feature_flag" "number" {
	project_key = launchdarkly_project.test.key
	key         = "numeric-flag"
	name        = "Number feature flag"
  
	variation_type = "number"
	variations = [{
	  value = 12.5
	}, {
	  value = 0
	}]
  }
`
	testAccFeatureFlagJsonBasic = `
resource "launchdarkly_feature_flag" "json_basic" {
	project_key = launchdarkly_project.test.key
	key         = "json-flag-basic"
	name        = "Basic JSON feature flag"
  
	variation_type = "json"
	variations = [{
	  value = <<EOF
	  {"foo": "bar"}
	  EOF
	}, {
	  value = <<EOF
	  {
		"bar": "foo",
		"bars": "foos"
	  }
	  EOF
	}]
  }
`

	testAccFeatureFlagJson = `
resource "launchdarkly_feature_flag" "json" {
	project_key = launchdarkly_project.test.key
	key         = "json-flag"
	name        = "JSON feature flag"
  
	variation_type = "json"
	variations = [{
	  value = <<EOF
	  [
		"foo",
		"baz"
	  ]
	  EOF
	}, {
	  value = <<EOF
	  {"foo": "bar"}
	  EOF
	}, {
	  value = <<EOF
	  {
		"foo": "baz",
		"extra": {"nested": "json"}
	  }
	  EOF
	}, {
		value = <<EOF
		{
		  "foo": ["nested", "array"]
		}
		EOF
	  }]
  }
`

	testAccFeatureFlagWithTeamMaintainer = `
	resource "launchdarkly_team_member" "test" {
		email = "%s+wbteste2e@launchdarkly.com"
		first_name = "first"
		last_name = "last"
		role = "admin"
		custom_roles = []
	}

	resource "launchdarkly_team" "test_team" {
		key                   = "%s"
		name                  = "test team"
		description           = "Team to manage team project"
		member_ids            = [launchdarkly_team_member.test.id]
		custom_role_keys      = []
	}

	resource "launchdarkly_feature_flag" "maintained" {
		project_key = launchdarkly_project.test.key
		key = "maintained-flag"
		name = "Maintained feature flag"
		variation_type = "boolean"
		variations = [
			{ value = "true" },
			{ value = "false" },
		]
		maintainer_team_key = launchdarkly_team.test_team.key
	}
	`

	// The email must be set with a random name using fmt.Sprintf for this test to work since LD does
	// not support creating members with the same email address more than once.
	testAccFeatureFlagWithMaintainer = `
resource "launchdarkly_team_member" "test" {
	email = "%s+wbteste2e@launchdarkly.com"
	first_name = "first"
	last_name = "last"
	role = "admin"
	custom_roles = []
}

resource "launchdarkly_feature_flag" "maintained" {
	project_key = launchdarkly_project.test.key
	key = "maintained-flag"
	name = "Maintained feature flag"
	variation_type = "boolean"
	variations = [
		{ value = "true" },
		{ value = "false" },
	]
	maintainer_id = launchdarkly_team_member.test.id
}
`

	// if the maintainer id is removed from the config it should still be set in the state to
	// the previous maintainer if that maintainer still exists
	testAccFeatureFlagMaintainerComputed = `
resource "launchdarkly_team_member" "test" {
	email = "%s+wbteste2e@launchdarkly.com"
	first_name = "first"
	last_name = "last"
	role = "admin"
	custom_roles = []
}

resource "launchdarkly_feature_flag" "maintained" {
	project_key = launchdarkly_project.test.key
	key = "maintained-flag"
	name = "Maintained feature flag"
	variation_type = "boolean"
	variations = [
		{ value = "true" },
		{ value = "false" },
	]
}
`

	//testAccFeatureFlagMaintainerDeleted is used to test that feature flag maintainers can be unset
	testAccFeatureFlagMaintainerDeleted = `
resource "launchdarkly_feature_flag" "maintained" {
	project_key = launchdarkly_project.test.key
	key = "maintained-flag"
	name = "Maintained feature flag"
	variation_type = "boolean"
	variations = [
		{ value = "true" },
		{ value = "false" },
	]
}
`

	testAccFeatureFlagWithInvalidMaintainer = `
resource "launchdarkly_feature_flag" "maintained" {
	project_key = launchdarkly_project.test.key
	key = "maintained-flag"
	name = "Maintained feature flag"
	variation_type = "boolean"
	variations = [
		{ value = "true" },
		{ value = "false" },
	]

	# the maintainer id set to a random object ID, so it should be invalid
	maintainer_id = "507f191e810c19729de860ea"
}
`

	testAccFeatureFlagCreateMultivariate = `
resource "launchdarkly_feature_flag" "multivariate" {
	project_key = launchdarkly_project.test.key
	key = "multivariate-flag-1"
	name = "multivariate flag 1 name"
	description = "this is a multivariate flag because we explicitly define the variations"
	variation_type = "string"
	variations = [
		{
			name        = "variation1"
			description = "a description"
			value       = "string1"
		},
		{ value = "string2" },
		{ value = "another option" },
	]
  	tags = [
    	"this",
    	"is",
    	"unordered"
  	]
  	custom_properties = [
		{
			key   = "some.property"
			name  = "Some Property"
			value = ["value1", "value2", "value3"]
		},
		{
			key   = "some.property2"
			name  = "Some Property"
			value = ["very special custom property"]
		},
	]
}
`

	testAccFeatureFlagCreateMultivariate2 = `
resource "launchdarkly_feature_flag" "multivariate_numbers" {
	project_key = launchdarkly_project.test.key
	key = "multivariate-flag-2"
	name = "multivariate flag 2 name"
	description = "this is a multivariate flag to test big number values"
	variation_type = "number"
	variations = [
		{
			name        = "variation1"
			description = "a description"
			value       = 86400000
		},
		{
			value = 123
		},
		{
			value = 123456789
		},
	]
  	tags = [
    	"this",
    	"is",
    	"unordered"
  	]
}
`

	testAccFeatureFlagUpdateMultivariate = `
resource "launchdarkly_feature_flag" "multivariate" {
	project_key = launchdarkly_project.test.key
	key = "multivariate-flag-1"
	name = "multivariate flag 1 name"
	description = "this is a multivariate flag because we explicitly define the variations"
	variation_type = "string"
	variations = [{
		name = "variation1"
		description = "a description"
		value = "string1"
	}, {
		value = "string2"
		description = "a new description"
	}, {
		value = "another option"
	}, {
		value = "a new variation"
		description = "This one was added upon update"
		name = "the new variation"
	}]
  	tags = [
    	"this",
    	"is",
    	"unordered"
  	]
  	custom_properties = [{
		key = "some.property"
		name = "Some Property Updated"
		value = [
			"value1",
			"value3"
		]
	}]
	defaults = [{
		on_variation = 2
		off_variation = 1
	}]
}
`

	testAccFeatureFlagDefaults = `
resource "launchdarkly_feature_flag" "defaults" {
	project_key = launchdarkly_project.test.key
	key = "defaults-flag"
	name = "Feature flag with defaults"
	variation_type = "boolean"
	variations = [
		{ value = "true" },
		{ value = "false" },
	]
	defaults = [{
		on_variation = 1
		off_variation = 1
	}]
}
`
	testAccFeatureFlagDefaultsUpdate = `
resource "launchdarkly_feature_flag" "defaults" {
	project_key = launchdarkly_project.test.key
	key = "defaults-flag"
	name = "Feature flag with defaults"
	variation_type = "boolean"
	variations = [
		{ value = "true" },
		{ value = "false" },
	]
	defaults = [{
		on_variation = 0
		off_variation = 0
	}]
}
`
	testAccFeatureFlagDefaultsMultivariate = `
resource "launchdarkly_feature_flag" "defaults-multivariate" {
	project_key = launchdarkly_project.test.key
	key = "defaults-multivariate-flag"
	name = "Multivariate feature flag with defaults"
	variation_type = "string"
	defaults = [{
		on_variation = 1
		off_variation = 1
	}]
	variations = [{
		value = "a"
	}, {
		value = "b"
	}, {
		value = "c"
	}, {
		value = "d"
	}]
}
`
	testAccFeatureFlagDefaultsMultivariateUpdate = `
resource "launchdarkly_feature_flag" "defaults-multivariate" {
	project_key = launchdarkly_project.test.key
	key = "defaults-multivariate-flag"
	name = "Multivariate feature flag with defaults"
	variation_type = "string"
	defaults = [{
		on_variation = 2
		off_variation = 2
	}]
	variations = [{
		value = "a"
	}, {
		value = "b"
	}, {
		value = "c"
	}, {
		value = "d"
	}]
}
`
	testAccFeatureFlagDefaultsMultivariateUpdateRemoveVariation = `
resource "launchdarkly_feature_flag" "defaults-multivariate" {
	project_key = launchdarkly_project.test.key
	key = "defaults-multivariate-flag"
	name = "Multivariate fature flag with defaults"
	variation_type = "string"
	defaults = [{
		on_variation = 2
		off_variation = 2
	}]
	variations = [{
		value = "b"
	}, {
		value = "c"
	}, {
		value = "d"
	}]
}
`
	testAccFeatureFlagEmptyStringVariation = `
resource "launchdarkly_feature_flag" "empty_string_variation" {
	project_key = launchdarkly_project.test.key
	key = "empty-variation"
	name = "string flag with empty string variation"
	variation_type = "string"
	variations = [{
		value = ""
	}, {
		value = "non-empty"
	}]
}
`
	testAccFeatureFlagIncludeInSnippet = `
resource "launchdarkly_feature_flag" "sdk_settings" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag-sdk-settings"
	name = "Basic feature flag"
	variation_type = "boolean"
	variations = [
		{ value = "true" },
		{ value = "false" },
	]
	include_in_snippet = true
}
`
	testAccFeatureFlagIncludeInSnippetUpdate = `
resource "launchdarkly_feature_flag" "sdk_settings" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag-sdk-settings"
	name = "Basic feature flag"
	variation_type = "boolean"
	variations = [
		{ value = "true" },
		{ value = "false" },
	]
	include_in_snippet = false
}
`
	testAccFeatureFlagIncludeInSnippetEmpty = `
resource "launchdarkly_feature_flag" "sdk_settings" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag-sdk-settings"
	name = "Basic feature flag"
	variation_type = "boolean"
	variations = [
		{ value = "true" },
		{ value = "false" },
	]
}
`
	testAccFeatureFlagClientSideAvailability = `
resource "launchdarkly_feature_flag" "sdk_settings" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag-sdk-settings"
	name = "Basic feature flag"
	variation_type = "boolean"
	variations = [
		{ value = "true" },
		{ value = "false" },
	]
	client_side_availability = [{
		using_environment_id = true
		using_mobile_key = true
	}]
}
`
	testAccFeatureFlagClientSideAvailabilityUpdate = `
resource "launchdarkly_feature_flag" "sdk_settings" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag-sdk-settings"
	name = "Basic feature flag"
	variation_type = "boolean"
	variations = [
		{ value = "true" },
		{ value = "false" },
	]
	client_side_availability = [{
		using_environment_id = false
		using_mobile_key = false
	}]
}
`
)

func withRandomProject(randomProject, resource string) string {
	return fmt.Sprintf(`
	resource "launchdarkly_project" "test" {
		lifecycle {
			ignore_changes = [environments]
		}
		name = "testProject"
		key = "%s"
		environments = [{
			name  = "testEnvironment"
			key   = "test"
			color = "000000"
		}]
	}
	
	%s`, randomProject, resource)
}

func withProjectWithSpecifiedCSADefaults(randomProject string, resource string, usingEnvironmentId bool, usingMobileKey bool) string {
	return fmt.Sprintf(`
	resource "launchdarkly_project" "test" {
		lifecycle {
			ignore_changes = [environments]
		}
		name = "testProject"
		key = "%s"
		default_client_side_availability = [{
			using_environment_id = %v
			using_mobile_key = %v
		}]
		environments = [{
			name  = "testEnvironment"
			key   = "test"
			color = "000000"
		}]
	}
	
	%s`, randomProject, usingEnvironmentId, usingMobileKey, resource)
}

func withRandomProjectAndEnv(randomProject, randomEnvironment, resource string) string {
	return fmt.Sprintf(`
	resource "launchdarkly_project" "test" {
		lifecycle {
			ignore_changes = [environments]
		}
		name = "testProject"
		key = "%s"
		environments = [{
			name  = "testEnvironment"
			key   = "%s"
			color = "000000"
		}]
	}
	
	%s`, randomProject, randomEnvironment, resource)
}

func withRandomProjectIncludeInSnippetTrue(randomProject, resource string) string {
	return fmt.Sprintf(`
	resource "launchdarkly_project" "test" {
		lifecycle {
			ignore_changes = [environments]
		}
		include_in_snippet = true
		name = "testProject"
		key = "%s"
		environments = [{
			name  = "testEnvironment"
			key   = "test"
			color = "000000"
		}]
	}
	
	%s`, randomProject, resource)
}

func TestAccFeatureFlag_BasicCreateAndUpdate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.basic"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagBasic),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "basic-flag"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, VARIATION_TYPE, "boolean"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "2"),
					resource.TestCheckNoResourceAttr(resourceName, "defaults.#"),
					resource.TestCheckNoResourceAttr(resourceName, MAINTAINER_ID),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Less basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "basic-flag"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "this is a boolean flag by default because the variations field is omitted"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "update"),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_IN_SNIPPET, "true"),
					resource.TestCheckResourceAttr(resourceName, TEMPORARY, "true"),
					resource.TestCheckResourceAttr(resourceName, "defaults.0.on_variation", "1"),
					resource.TestCheckResourceAttr(resourceName, "defaults.0.off_variation", "1"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
		},
	})
}

func TestAccFeatureFlag_CSAInteractionWithProjectDefaults(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.basic"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagBasic),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "basic-flag"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, VARIATION_TYPE, "boolean"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "2"),
					resource.TestCheckNoResourceAttr(resourceName, "defaults.#"),
					resource.TestCheckNoResourceAttr(resourceName, "client_side_availability.#"),
					resource.TestCheckNoResourceAttr(resourceName, MAINTAINER_ID),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
			{
				Config: withProjectWithSpecifiedCSADefaults(projectKey, testAccFeatureFlagBasicWithTag, false, false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "basic-flag"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, VARIATION_TYPE, "boolean"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "2"),
					resource.TestCheckNoResourceAttr(resourceName, "defaults.#"),
					resource.TestCheckNoResourceAttr(resourceName, "client_side_availability.#"),
					resource.TestCheckNoResourceAttr(resourceName, MAINTAINER_ID),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
			{
				Config: withProjectWithSpecifiedCSADefaults(projectKey, testAccFeatureFlagBasicWithCSASet, false, false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "basic-flag"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, VARIATION_TYPE, "boolean"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "2"),
					resource.TestCheckNoResourceAttr(resourceName, "defaults.#"),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_environment_id", "true"),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_mobile_key", "true"),
					resource.TestCheckNoResourceAttr(resourceName, MAINTAINER_ID),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
			{
				Config: withProjectWithSpecifiedCSADefaults(projectKey, testAccFeatureFlagBasicWithTag, false, false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "basic-flag"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, VARIATION_TYPE, "boolean"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "2"),
					resource.TestCheckNoResourceAttr(resourceName, "defaults.#"),
					// Once the user omits client_side_availability, state drops
					// it back to null rather than retaining the last set value.
					resource.TestCheckNoResourceAttr(resourceName, "client_side_availability.#"),
					resource.TestCheckNoResourceAttr(resourceName, MAINTAINER_ID),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
		},
	})
}

func TestAccFeatureFlag_Number(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.number"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagNumber),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Number feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "numeric-flag"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, VARIATION_TYPE, "number"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.value", "12.5"),
					resource.TestCheckResourceAttr(resourceName, "variations.1.value", "0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// Import has no prior plan/state, so framework Read can't
				// tell user-declared from omitted Optional+Computed attrs.
				// Suppress the resulting pre/post divergence for these
				// paths.
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
		},
	})
}

// importIgnoreOptionalComputedKeys is the canonical set of attribute
// paths suppressed by ImportStateVerify for tests that declare any of
// variations / defaults / client_side_availability — Import emits the
// API's view of these and diverges from the user's config-anchored
// state.
var importIgnoreOptionalComputedKeys = []string{
	"variations.#",
	"variations.0.%", "variations.0.value", "variations.0.name", "variations.0.description",
	"variations.1.%", "variations.1.value", "variations.1.name", "variations.1.description",
	"variations.2.%", "variations.2.value", "variations.2.name", "variations.2.description",
	"variations.3.%", "variations.3.value", "variations.3.name", "variations.3.description",
	"defaults.#",
	"defaults.0.%", "defaults.0.on_variation", "defaults.0.off_variation",
	"client_side_availability.#",
	"client_side_availability.0.%",
	"client_side_availability.0.using_environment_id", "client_side_availability.0.using_mobile_key",
}

func TestAccFeatureFlag_JSONBasic(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.json_basic"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagJsonBasic),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Basic JSON feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "json-flag-basic"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, VARIATION_TYPE, "json"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "2"),
					testCheckJSONVariationEqual(resourceName, "variations.0.value", `{"foo":"bar"}`),
					testCheckJSONVariationEqual(resourceName, "variations.1.value", `{"bar":"foo","bars":"foos"}`),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
		},
	})
}
func TestAccFeatureFlag_JSON(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.json"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagJson),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "JSON feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "json-flag"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, VARIATION_TYPE, "json"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "4"),
					testCheckJSONVariationEqual(resourceName, "variations.0.value", `["foo","baz"]`),
					testCheckJSONVariationEqual(resourceName, "variations.1.value", `{"foo":"bar"}`),
					testCheckJSONVariationEqual(resourceName, "variations.2.value", `{"extra":{"nested":"json"},"foo":"baz"}`),
					testCheckJSONVariationEqual(resourceName, "variations.3.value", `{"foo":["nested","array"]}`),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
		},
	})
}

// testCheckJSONVariationEqual asserts that the state attribute at the
// given path is semantically equal to the expected JSON, regardless of
// whitespace formatting. SDKv2 stored variation values in the API's
// compact canonical form because of how its read path serialised the
// `interface{}`-typed value back to a string; the plugin framework's
// variationsListFromAPI preserves the user-provided HCL representation
// to keep plan and state byte-equal (terraform-core enforces
// plan == config for Required attributes). The asserter only needs
// JSON equivalence.
func testCheckJSONVariationEqual(resourceName, key, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found", resourceName)
		}
		got, ok := rs.Primary.Attributes[key]
		if !ok {
			return fmt.Errorf("attribute %s not found on %s", key, resourceName)
		}
		var gotJSON, expectedJSON interface{}
		if err := json.Unmarshal([]byte(got), &gotJSON); err != nil {
			return fmt.Errorf("%s.%s: state value is not valid JSON: %s", resourceName, key, got)
		}
		if err := json.Unmarshal([]byte(expected), &expectedJSON); err != nil {
			return fmt.Errorf("%s.%s: expected value is not valid JSON: %s", resourceName, key, expected)
		}
		if !reflect.DeepEqual(gotJSON, expectedJSON) {
			return fmt.Errorf("%s.%s: JSON values differ\n  expected: %s\n  got:      %s", resourceName, key, expected, got)
		}
		return nil
	}
}

func TestAccFeatureFlag_WithMaintainer(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	randomName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.maintained"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, fmt.Sprintf(testAccFeatureFlagWithTeamMaintainer, randomName, randomName)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckMemberExists("launchdarkly_team_member.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Maintained feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "maintained-flag"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, MAINTAINER_ID, ""),
					resource.TestCheckResourceAttrPair(resourceName, MAINTAINER_TEAM_KEY, "launchdarkly_team.test_team", "id"),
					resource.TestCheckResourceAttrPair(resourceName, MAINTAINER_TEAM_KEY, "launchdarkly_team.test_team", "key"),
				),
			},
			{
				Config: withRandomProject(projectKey, fmt.Sprintf(testAccFeatureFlagWithMaintainer, randomName)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckMemberExists("launchdarkly_team_member.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Maintained feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "maintained-flag"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttrPair(resourceName, MAINTAINER_ID, "launchdarkly_team_member.test", "id"),
					resource.TestCheckResourceAttr(resourceName, MAINTAINER_TEAM_KEY, ""),
				),
			},
			{
				ResourceName: resourceName,
				ImportState:  true,
				// ImportStateVerify: true, // this is broken on this test
			},
			{
				Config: withRandomProject(projectKey, fmt.Sprintf(testAccFeatureFlagMaintainerComputed, randomName)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Maintained feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "maintained-flag"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					// when removed it should reset back to the most recently-set maintainer
					resource.TestCheckResourceAttrPair(resourceName, MAINTAINER_ID, "launchdarkly_team_member.test", "id"),
				),
			},
			{
				ResourceName: resourceName,
				ImportState:  true,
				// ImportStateVerify: true,
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagMaintainerDeleted),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Maintained feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "maintained-flag"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					// it will still be set to the most recently set one even if that member has been deleted
					// the UI will not show a maintainer because it will not be able to find the record post-member delete
					resource.TestCheckResourceAttrSet(resourceName, MAINTAINER_ID),
				),
			},
			{
				ResourceName: resourceName,
				ImportState:  true,
				// ImportStateVerify: true,
			},
		},
	})
}

// TestAccFeatureFlag_WithInvalidMaintainer tests that flags that fail during the update portion of the create clean up
// after themselves and do not leave dangling flags.
func TestAccFeatureFlag_InvalidMaintainer(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	randomName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.maintained"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      withRandomProject(projectKey, testAccFeatureFlagWithInvalidMaintainer),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`failed to update flag "maintained-flag" in project "%s": 400 Bad Request`, projectKey)),
			},
			{
				Config: withRandomProject(projectKey, fmt.Sprintf(testAccFeatureFlagWithMaintainer, randomName)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckMemberExists("launchdarkly_team_member.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Maintained feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "maintained-flag"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttrPair(resourceName, MAINTAINER_ID, "launchdarkly_team_member.test", "id"),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagMaintainerDeleted),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Maintained feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "maintained-flag"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					// this is the best we can do. it should default back to the most recently-set maintainer but
					// we have no easy way of a
					resource.TestCheckResourceAttrSet(resourceName, MAINTAINER_ID),
				),
			},
		},
	})
}

func TestAccFeatureFlag_CreateAndUpdateMultivariate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.multivariate"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagCreateMultivariate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "multivariate flag 1 name"),
					resource.TestCheckResourceAttr(resourceName, KEY, "multivariate-flag-1"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "this is a multivariate flag because we explicitly define the variations"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.description", "a description"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.name", "variation1"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.value", "string1"),
					resource.TestCheckResourceAttr(resourceName, "variations.1.value", "string2"),
					resource.TestCheckResourceAttr(resourceName, "variations.2.value", "another option"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "3"),
					// the v2 terraform sdk forces you to index TypeSet attributes like tags on an ordered index
					resource.TestCheckResourceAttr(resourceName, "tags.0", "is"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "this"),
					resource.TestCheckResourceAttr(resourceName, "tags.2", "unordered"),
					resource.TestCheckResourceAttr(resourceName, "custom_properties.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "custom_properties.0.key", "some.property"),
					resource.TestCheckResourceAttr(resourceName, "custom_properties.0.name", "Some Property"),
					resource.TestCheckResourceAttr(resourceName, "custom_properties.0.value.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "custom_properties.0.value.0", "value1"),
					resource.TestCheckResourceAttr(resourceName, "custom_properties.0.value.1", "value2"),
					resource.TestCheckResourceAttr(resourceName, "custom_properties.0.value.2", "value3"),
					resource.TestCheckResourceAttr(resourceName, "custom_properties.1.key", "some.property2"),
					resource.TestCheckResourceAttr(resourceName, "custom_properties.1.name", "Some Property"),
					resource.TestCheckResourceAttr(resourceName, "custom_properties.1.value.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_properties.1.value.0", "very special custom property"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagUpdateMultivariate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "multivariate flag 1 name"),
					resource.TestCheckResourceAttr(resourceName, KEY, "multivariate-flag-1"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "this is a multivariate flag because we explicitly define the variations"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "4"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.description", "a description"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.name", "variation1"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.value", "string1"),
					resource.TestCheckResourceAttr(resourceName, "variations.1.value", "string2"),
					resource.TestCheckResourceAttr(resourceName, "variations.1.description", "a new description"),
					resource.TestCheckResourceAttr(resourceName, "variations.2.value", "another option"),
					resource.TestCheckResourceAttr(resourceName, "variations.3.value", "a new variation"),
					resource.TestCheckResourceAttr(resourceName, "variations.3.name", "the new variation"),
					resource.TestCheckResourceAttr(resourceName, "variations.3.description", "This one was added upon update"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "is"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "this"),
					resource.TestCheckResourceAttr(resourceName, "tags.2", "unordered"),
					resource.TestCheckResourceAttr(resourceName, "custom_properties.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_properties.0.key", "some.property"),
					resource.TestCheckResourceAttr(resourceName, "custom_properties.0.name", "Some Property Updated"),
					resource.TestCheckResourceAttr(resourceName, "custom_properties.0.value.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "custom_properties.0.value.0", "value1"),
					resource.TestCheckResourceAttr(resourceName, "custom_properties.0.value.1", "value3"),
					resource.TestCheckResourceAttr(resourceName, "defaults.0.on_variation", "2"),
					resource.TestCheckResourceAttr(resourceName, "defaults.0.off_variation", "1"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
			{
				// Ensure variation Delete operations are working
				Config: withRandomProject(projectKey, testAccFeatureFlagCreateMultivariate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.description", "a description"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.name", "variation1"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.value", "string1"),
					resource.TestCheckResourceAttr(resourceName, "variations.1.value", "string2"),
					resource.TestCheckResourceAttr(resourceName, "variations.2.value", "another option"),
					// defaults is Optional-only and omitted here, so state
					// drops it rather than retaining the previous-step value.
					resource.TestCheckNoResourceAttr(resourceName, "defaults.#"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
		},
	})
}

func TestAccFeatureFlag_CreateMultivariate2(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.multivariate_numbers"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagCreateMultivariate2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "multivariate flag 2 name"),
					resource.TestCheckResourceAttr(resourceName, KEY, "multivariate-flag-2"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "this is a multivariate flag to test big number values"),
					resource.TestCheckResourceAttr(resourceName, VARIATION_TYPE, "number"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.description", "a description"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.name", "variation1"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.value", "86400000"),
					resource.TestCheckResourceAttr(resourceName, "variations.1.value", "123"),
					resource.TestCheckResourceAttr(resourceName, "variations.2.value", "123456789"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "is"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "this"),
					resource.TestCheckResourceAttr(resourceName, "tags.2", "unordered"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
		},
	})
}

func TestAccFeatureFlag_UpdateDefaults(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.defaults"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagDefaults),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "defaults.0.on_variation", "1"),
					resource.TestCheckResourceAttr(resourceName, "defaults.0.off_variation", "1"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagDefaultsUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "defaults.0.on_variation", "0"),
					resource.TestCheckResourceAttr(resourceName, "defaults.0.off_variation", "0"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
		},
	})
}

func TestAccFeatureFlag_UpdateMultivariateDefaults(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.defaults-multivariate"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagDefaultsMultivariate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "defaults.0.on_variation", "1"),
					resource.TestCheckResourceAttr(resourceName, "defaults.0.off_variation", "1"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagDefaultsMultivariateUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "defaults.0.on_variation", "2"),
					resource.TestCheckResourceAttr(resourceName, "defaults.0.off_variation", "2"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagDefaultsMultivariateUpdateRemoveVariation),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "defaults.0.on_variation", "2"),
					resource.TestCheckResourceAttr(resourceName, "defaults.0.off_variation", "2"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
		},
	})
}

func TestAccFeatureFlag_EmptyStringVariation(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.empty_string_variation"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagEmptyStringVariation),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "variations.0.value", ""),
					resource.TestCheckResourceAttr(resourceName, "variations.0.name", ""),
					resource.TestCheckResourceAttr(resourceName, "variations.0.description", ""),
					resource.TestCheckResourceAttr(resourceName, "variations.1.value", "non-empty"),
					resource.TestCheckResourceAttr(resourceName, "variations.1.name", ""),
					resource.TestCheckResourceAttr(resourceName, "variations.1.description", ""),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
		},
	})
}

func TestAccFeatureFlag_ClientSideAvailabilityUpdate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.sdk_settings"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagClientSideAvailability),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "basic-flag-sdk-settings"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, VARIATION_TYPE, "boolean"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "2"),
					resource.TestCheckNoResourceAttr(resourceName, MAINTAINER_ID),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_environment_id", "true"),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_mobile_key", "true"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagClientSideAvailabilityUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "basic-flag-sdk-settings"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, VARIATION_TYPE, "boolean"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "2"),
					resource.TestCheckNoResourceAttr(resourceName, MAINTAINER_ID),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_environment_id", "false"),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_mobile_key", "false"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
		},
	})
}

func TestAccFeatureFlag_IncludeInSnippetToClientSide(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.sdk_settings"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagIncludeInSnippet),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "basic-flag-sdk-settings"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, VARIATION_TYPE, "boolean"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "2"),
					resource.TestCheckNoResourceAttr(resourceName, "client_side_availability.#"),
					resource.TestCheckNoResourceAttr(resourceName, MAINTAINER_ID),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_IN_SNIPPET, "true"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagClientSideAvailability),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "basic-flag-sdk-settings"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, VARIATION_TYPE, "boolean"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "2"),
					resource.TestCheckNoResourceAttr(resourceName, MAINTAINER_ID),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_environment_id", "true"),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_mobile_key", "true"),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_IN_SNIPPET, "true"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagClientSideAvailabilityUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "basic-flag-sdk-settings"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, VARIATION_TYPE, "boolean"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "2"),
					resource.TestCheckNoResourceAttr(resourceName, MAINTAINER_ID),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_environment_id", "false"),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_mobile_key", "false"),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_IN_SNIPPET, "false"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
		},
	})
}

func TestAccFeatureFlag_ClientSideToIncludeInSnippet(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.sdk_settings"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagClientSideAvailability),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "basic-flag-sdk-settings"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, VARIATION_TYPE, "boolean"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "2"),
					resource.TestCheckNoResourceAttr(resourceName, MAINTAINER_ID),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_environment_id", "true"),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_mobile_key", "true"),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_IN_SNIPPET, "true"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagIncludeInSnippetUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "basic-flag-sdk-settings"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, VARIATION_TYPE, "boolean"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "2"),
					resource.TestCheckNoResourceAttr(resourceName, MAINTAINER_ID),
					// Removing client_side_availability drops it from state
					// rather than retaining the previously-set values.
					resource.TestCheckNoResourceAttr(resourceName, "client_side_availability.#"),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_IN_SNIPPET, "false"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
		},
	})
}

func TestAccFeatureFlag_IncludeInSnippet(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.sdk_settings"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create without value set and check for default value
			{
				Config: withRandomProjectIncludeInSnippetTrue(projectKey, testAccFeatureFlagIncludeInSnippetEmpty),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "basic-flag-sdk-settings"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_IN_SNIPPET, "true"),
					// Omitted client_side_availability stays null in state;
					// project defaults are not inflated onto the flag.
					resource.TestCheckNoResourceAttr(resourceName, "client_side_availability.#"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
			// Replace default value with specific value
			{
				Config: withRandomProjectIncludeInSnippetTrue(projectKey, testAccFeatureFlagIncludeInSnippetUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "basic-flag-sdk-settings"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_IN_SNIPPET, "false"),
					resource.TestCheckNoResourceAttr(resourceName, "client_side_availability.#"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
			// Clear specific value, should not revert to default
			{
				Config: withRandomProjectIncludeInSnippetTrue(projectKey, testAccFeatureFlagIncludeInSnippetEmpty),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "basic-flag-sdk-settings"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_IN_SNIPPET, "false"),
					resource.TestCheckNoResourceAttr(resourceName, "client_side_availability.#"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
		},
	})
}

func testAccCheckFeatureFlagExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		flagKey, ok := rs.Primary.Attributes[KEY]
		if !ok {
			return fmt.Errorf("feature flag key not found: %s", resourceName)
		}
		projKey, ok := rs.Primary.Attributes[PROJECT_KEY]
		if !ok {
			return fmt.Errorf("project key not found: %s", resourceName)
		}
		client := mustTestAccClient()
		_, _, err := client.ld.FeatureFlagsApi.GetFeatureFlag(client.ctx, projKey, flagKey).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting feature flag. %s", err)
		}
		return nil
	}
}

// TestAccFeatureFlag_Deprecated tests that the deprecated attribute is set correctly
// fails when the property is not set correctly
func TestAccFeatureFlag_Deprecated(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.deprecated"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagDeprecated),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, DEPRECATED, "true"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagUndeprecated),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, DEPRECATED, "false"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: importIgnoreOptionalComputedKeys,
			},
		},
	})
}

// TestAccFeatureFlag_ViewAssociationRequired tests that creating a flag without view_keys
// fails when the project requires view association for new flags
func TestAccFeatureFlag_ViewAssociationRequired(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.test"
	testAccFlagWithViewKeys := ""

	// Config with project requiring view association but flag without view_keys (should fail)
	testAccFlagWithoutViewKeys := fmt.Sprintf(`
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "View Requirement Test"
	require_view_association_for_new_flags = true
	environments = [{
		key   = "test-env"
		name  = "Test Environment"
		color = "010101"
	}]
}

resource "launchdarkly_feature_flag" "test" {
	project_key    = launchdarkly_project.test.key
	key            = "test-flag-no-views"
	name           = "Test Flag Without Views"
	variation_type = "boolean"
	variations = [
		{ value = "true" },
		{ value = "false" },
	]
}
`, projectKey)

	// Config with project requiring view association and flag with view_keys (should succeed)
	testAccFlagWithViewKeysTemplate := `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "View Requirement Test"
	require_view_association_for_new_flags = true
	environments = [{
		key   = "test-env"
		name  = "Test Environment"
		color = "010101"
	}]
}

resource "launchdarkly_view" "test" {
	project_key   = launchdarkly_project.test.key
	key           = "test-view"
	name          = "Test View"
	maintainer_id = "%s"
}

resource "launchdarkly_feature_flag" "test" {
	project_key    = launchdarkly_project.test.key
	key            = "test-flag-with-views"
	name           = "Test Flag With Views"
	variation_type = "boolean"
	variations = [
		{ value = "true" },
		{ value = "false" },
	]
	view_keys      = [launchdarkly_view.test.key]
}
`
	maintainerID := "507f1f77bcf86cd799439011"
	if os.Getenv("TF_ACC") != "" {
		testAccPreCheck(t)
		maintainerID = firstMemberIDForTest(t)
	}
	testAccFlagWithViewKeys = fmt.Sprintf(testAccFlagWithViewKeysTemplate, projectKey, maintainerID)

	// Config with project NOT requiring view association and flag without view_keys (should succeed)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckProjectDestroy,
		Steps: []resource.TestStep{
			// Step 1: Verify flag without view_keys fails when project requires it
			{
				Config:      testAccFlagWithoutViewKeys,
				ExpectError: regexp.MustCompile(`requires new flags to be associated with at least one view`),
			},
			// Step 2: Verify flag with view_keys succeeds when project requires it
			{
				Config: testAccFlagWithViewKeys,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "view_keys.#", "1"),
				),
			},
		},
	})
}
