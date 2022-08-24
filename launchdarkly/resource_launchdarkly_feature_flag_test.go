package launchdarkly

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	testAccFeatureFlagBasic = `
resource "launchdarkly_feature_flag" "basic" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag"
	name = "Basic feature flag"
	variation_type = "boolean"
}
`
	testAccFeatureFlagUpdate = `
resource "launchdarkly_feature_flag" "basic" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag"
	name = "Less basic feature flag"
	variation_type = "boolean"
	description = "this is a boolean flag by default becausethe variations field is omitted"
	tags = ["update", "terraform"]
	include_in_snippet = true
	temporary = true
	defaults {
		on_variation = 0
		off_variation = 1
	}
}
`

	testAccFeatureFlagNumber = `
resource "launchdarkly_feature_flag" "number" {
	project_key = launchdarkly_project.test.key
	key         = "numeric-flag"
	name        = "Number feature flag"
  
	variation_type = "number"
	variations {
	  value = 12.5
	}
	variations {
	  value = 0
	}
  }
`
	testAccFeatureFlagJsonBasic = `
resource "launchdarkly_feature_flag" "json_basic" {
	project_key = launchdarkly_project.test.key
	key         = "json-flag-basic"
	name        = "Basic JSON feature flag"
  
	variation_type = "json"
	variations {
	  value = <<EOF
	  {"foo": "bar"}
	  EOF
	}
	variations {
	  value = <<EOF
	  {
		"bar": "foo",
		"bars": "foos"
	  }
	  EOF
	}
  }
`

	testAccFeatureFlagJson = `
resource "launchdarkly_feature_flag" "json" {
	project_key = launchdarkly_project.test.key
	key         = "json-flag"
	name        = "JSON feature flag"
  
	variation_type = "json"
	variations {
	  value = <<EOF
	  [
		"foo",
		"baz"
	  ]
	  EOF
	}
	variations {
	  value = <<EOF
	  {"foo": "bar"}
	  EOF
	}
	variations {
	  value = <<EOF
	  {
		"foo": "baz",
		"extra": {"nested": "json"}
	  }
	  EOF
	}
	variations {
		value = <<EOF
		{
		  "foo": ["nested", "array"]
		}
		EOF
	  }
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
}
`

	//testAccFeatureFlagMaintainerDeleted is used to test that feature flag maintainers can be unset
	testAccFeatureFlagMaintainerDeleted = `
resource "launchdarkly_feature_flag" "maintained" {
	project_key = launchdarkly_project.test.key
	key = "maintained-flag"
	name = "Maintained feature flag"
	variation_type = "boolean"
}
`

	testAccFeatureFlagWithInvalidMaintainer = `
resource "launchdarkly_feature_flag" "maintained" {
	project_key = launchdarkly_project.test.key
	key = "maintained-flag"
	name = "Maintained feature flag"
	variation_type = "boolean"

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
	variations {
		name = "variation1"
		description = "a description"
		value = "string1"
	}
    variations {
		value = "string2"
	}
    variations {
		value = "another option"
	}
  	tags = [
    	"this",
    	"is",
    	"unordered"
  	]
  	custom_properties {
		key = "some.property"
		name = "Some Property"
		value = [
			"value1",
			"value2",
			"value3"
		]
	}
	custom_properties {
		key = "some.property2"
		name = "Some Property"
		value = ["very special custom property"]
	}
}
`

	testAccFeatureFlagCreateMultivariate2 = `
resource "launchdarkly_feature_flag" "multivariate_numbers" {
	project_key = launchdarkly_project.test.key
	key = "multivariate-flag-2"
	name = "multivariate flag 2 name"
	description = "this is a multivariate flag to test big number values"
	variation_type = "number"
	variations {
		name = "variation1"
		description = "a description"
		value = 86400000
	}
    variations {
		value = 123
	}
    variations {
		value = 123456789
	}
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
	variations {
		name = "variation1"
		description = "a description"
		value = "string1"
	}
	variations {
		value = "string2"
		description = "a new description"
	}
	variations {
		value = "another option"
	}
	variations {
		value = "a new variation"
		description = "This one was added upon update"
		name = "the new variation"
	}
  	tags = [
    	"this",
    	"is",
    	"unordered"
  	]
  	custom_properties {
		key = "some.property"
		name = "Some Property Updated"
		value = [
			"value1",
			"value3"
		]
	}
	defaults {
		on_variation = 2
		off_variation = 1
	}
}
`

	testAccFeatureFlagDefaults = `
resource "launchdarkly_feature_flag" "defaults" {
	project_key = launchdarkly_project.test.key
	key = "defaults-flag"
	name = "Feature flag with defaults"
	variation_type = "boolean"
	defaults {
		on_variation = 0
		off_variation = 1
	}
}
`
	testAccFeatureFlagDefaultsUpdate = `
resource "launchdarkly_feature_flag" "defaults" {
	project_key = launchdarkly_project.test.key
	key = "defaults-flag"
	name = "Feature flag with defaults"
	variation_type = "boolean"
	defaults {
		on_variation = 0
		off_variation = 0
	}
}
`
	testAccFeatureFlagDefaultsMultivariate = `
resource "launchdarkly_feature_flag" "defaults-multivariate" {
	project_key = launchdarkly_project.test.key
	key = "defaults-multivariate-flag"
	name = "Multivariate feature flag with defaults"
	variation_type = "string"
	defaults {
		on_variation = 1
		off_variation = 1
	}
	variations {
		value = "a"
	}
	variations {
		value = "b"
	}
	variations {
		value = "c"
	}
	variations {
		value = "d"
	}
}
`
	testAccFeatureFlagDefaultsMultivariateUpdate = `
resource "launchdarkly_feature_flag" "defaults-multivariate" {
	project_key = launchdarkly_project.test.key
	key = "defaults-multivariate-flag"
	name = "Multivariate feature flag with defaults"
	variation_type = "string"
	defaults {
		on_variation = 2
		off_variation = 2
	}
	variations {
		value = "a"
	}
	variations {
		value = "b"
	}
	variations {
		value = "c"
	}
	variations {
		value = "d"
	}
}
`
	testAccFeatureFlagDefaultsMultivariateUpdateRemoveVariation = `
resource "launchdarkly_feature_flag" "defaults-multivariate" {
	project_key = launchdarkly_project.test.key
	key = "defaults-multivariate-flag"
	name = "Multivariate fature flag with defaults"
	variation_type = "string"
	defaults {
		on_variation = 2
		off_variation = 2
	}
	variations {
		value = "b"
	}
	variations {
		value = "c"
	}
	variations {
		value = "d"
	}
}
`
	testAccFeatureFlagEmptyStringVariation = `
resource "launchdarkly_feature_flag" "empty_string_variation" {
	project_key = launchdarkly_project.test.key
	key = "empty-variation"
	name = "string flag with empty string variation"
	variation_type = "string"
	variations {
		value = ""
	}
	variations {
		value = "non-empty"
	}
}
`
	testAccFeatureFlagIncludeInSnippet = `
resource "launchdarkly_feature_flag" "sdk_settings" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag-sdk-settings"
	name = "Basic feature flag"
	variation_type = "boolean"
	include_in_snippet = true
}
`
	testAccFeatureFlagIncludeInSnippetUpdate = `
resource "launchdarkly_feature_flag" "sdk_settings" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag-sdk-settings"
	name = "Basic feature flag"
	variation_type = "boolean"
	include_in_snippet = false
}
`
	testAccFeatureFlagIncludeInSnippetEmpty = `
resource "launchdarkly_feature_flag" "sdk_settings" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag-sdk-settings"
	name = "Basic feature flag"
	variation_type = "boolean"
}
`
	testAccFeatureFlagClientSideAvailability = `
resource "launchdarkly_feature_flag" "sdk_settings" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag-sdk-settings"
	name = "Basic feature flag"
	variation_type = "boolean"
	client_side_availability {
		using_environment_id = true
		using_mobile_key = true
	}
}
`
	testAccFeatureFlagClientSideAvailabilityUpdate = `
resource "launchdarkly_feature_flag" "sdk_settings" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag-sdk-settings"
	name = "Basic feature flag"
	variation_type = "boolean"
	client_side_availability {
		using_environment_id = false
		using_mobile_key = false
	}
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
		environments {
			name  = "testEnvironment"
			key   = "test"
			color = "000000"
		}
	}
	
	%s`, randomProject, resource)
}

func withRandomProjectAndEnv(randomProject, randomEnvironment, resource string) string {
	return fmt.Sprintf(`
	resource "launchdarkly_project" "test" {
		lifecycle {
			ignore_changes = [environments]
		}
		name = "testProject"
		key = "%s"
		environments {
			name  = "testEnvironment"
			key   = "%s"
			color = "000000"
		}
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
		environments {
			name  = "testEnvironment"
			key   = "test"
			color = "000000"
		}
	}
	
	%s`, randomProject, resource)
}

func TestAccFeatureFlag_Basic(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.basic"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
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
					resource.TestCheckResourceAttr(resourceName, "variations.0.value", "true"),
					resource.TestCheckResourceAttr(resourceName, "variations.1.value", "false"),
					resource.TestCheckNoResourceAttr(resourceName, MAINTAINER_ID),
				),
			},
			{
				ResourceName: resourceName,
				ImportState:  true,
				// TODO: While we have to account for usingMobileKey being set to true by default, we cant use importStateVerify
				// ImportStateVerify: true,
			},
		},
	})
}

func TestAccFeatureFlag_Update(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.basic"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagBasic),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "basic-flag"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Less basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "basic-flag"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "this is a boolean flag by default becausethe variations field is omitted"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "update"),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_IN_SNIPPET, "true"),
					resource.TestCheckResourceAttr(resourceName, TEMPORARY, "true"),
					resource.TestCheckResourceAttr(resourceName, "defaults.0.on_variation", "0"),
					resource.TestCheckResourceAttr(resourceName, "defaults.0.off_variation", "1"),
				),
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
		Providers: testAccProviders,
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
				ResourceName: resourceName,
				ImportState:  true,
				// TODO: While we have to account for usingMobileKey being set to true by default, we cant use importStateVerify
				// ImportStateVerify: true,
			},
		},
	})
}

func TestAccFeatureFlag_JSONBasic(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.json_basic"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
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
					resource.TestCheckResourceAttr(resourceName, "variations.0.value", `{"foo":"bar"}`),
				),
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
		Providers: testAccProviders,
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
					resource.TestCheckResourceAttr(resourceName, "variations.0.value", `["foo","baz"]`),
					resource.TestCheckResourceAttr(resourceName, "variations.1.value", `{"foo":"bar"}`),
					resource.TestCheckResourceAttr(resourceName, "variations.2.value", `{"extra":{"nested":"json"},"foo":"baz"}`),
					resource.TestCheckResourceAttr(resourceName, "variations.3.value", `{"foo":["nested","array"]}`),
				),
			},
		},
	})
}

func TestAccFeatureFlag_WithMaintainer(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	randomName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.maintained"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
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
		Providers: testAccProviders,
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

func TestAccFeatureFlag_CreateMultivariate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.multivariate"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
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
		Providers: testAccProviders,
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
		},
	})
}

func TestAccFeatureFlag_UpdateMultivariate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.multivariate"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
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
					resource.TestCheckResourceAttr(resourceName, "defaults.0.on_variation", "2"),
					resource.TestCheckResourceAttr(resourceName, "defaults.0.off_variation", "1"),
				),
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
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagDefaults),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "defaults.0.on_variation", "0"),
					resource.TestCheckResourceAttr(resourceName, "defaults.0.off_variation", "1"),
				),
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
				ResourceName: resourceName,
				ImportState:  true,
				// TODO: While we have to account for usingMobileKey being set to true by default, we cant use importStateVerify
				// ImportStateVerify: true,
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
		Providers: testAccProviders,
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
				Config: withRandomProject(projectKey, testAccFeatureFlagDefaultsMultivariateUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "defaults.0.on_variation", "2"),
					resource.TestCheckResourceAttr(resourceName, "defaults.0.off_variation", "2"),
				),
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
				ResourceName: resourceName,
				ImportState:  true,
				// TODO: While we have to account for usingMobileKey being set to true by default, we cant use importStateVerify
				// ImportStateVerify: true,
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
		Providers: testAccProviders,
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
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
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
		Providers: testAccProviders,
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
					resource.TestCheckResourceAttr(resourceName, "variations.0.value", "true"),
					resource.TestCheckResourceAttr(resourceName, "variations.1.value", "false"),
					resource.TestCheckNoResourceAttr(resourceName, MAINTAINER_ID),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_environment_id", "true"),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_mobile_key", "true"),
				),
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
					resource.TestCheckResourceAttr(resourceName, "variations.0.value", "true"),
					resource.TestCheckResourceAttr(resourceName, "variations.1.value", "false"),
					resource.TestCheckNoResourceAttr(resourceName, MAINTAINER_ID),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_environment_id", "false"),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_mobile_key", "false"),
				),
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
		Providers: testAccProviders,
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
					resource.TestCheckResourceAttr(resourceName, "variations.0.value", "true"),
					resource.TestCheckResourceAttr(resourceName, "variations.1.value", "false"),
					resource.TestCheckNoResourceAttr(resourceName, MAINTAINER_ID),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_IN_SNIPPET, "true"),
				),
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
					resource.TestCheckResourceAttr(resourceName, "variations.0.value", "true"),
					resource.TestCheckResourceAttr(resourceName, "variations.1.value", "false"),
					resource.TestCheckNoResourceAttr(resourceName, MAINTAINER_ID),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_environment_id", "true"),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_mobile_key", "true"),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_IN_SNIPPET, "true"),
				),
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
					resource.TestCheckResourceAttr(resourceName, "variations.0.value", "true"),
					resource.TestCheckResourceAttr(resourceName, "variations.1.value", "false"),
					resource.TestCheckNoResourceAttr(resourceName, MAINTAINER_ID),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_environment_id", "false"),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_mobile_key", "false"),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_IN_SNIPPET, "false"),
				),
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
		Providers: testAccProviders,
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
					resource.TestCheckResourceAttr(resourceName, "variations.0.value", "true"),
					resource.TestCheckResourceAttr(resourceName, "variations.1.value", "false"),
					resource.TestCheckNoResourceAttr(resourceName, MAINTAINER_ID),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_environment_id", "true"),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_mobile_key", "true"),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_IN_SNIPPET, "true"),
				),
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
					resource.TestCheckResourceAttr(resourceName, "variations.0.value", "true"),
					resource.TestCheckResourceAttr(resourceName, "variations.1.value", "false"),
					resource.TestCheckNoResourceAttr(resourceName, MAINTAINER_ID),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_environment_id", "false"),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_mobile_key", "false"),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_IN_SNIPPET, "false"),
				),
			},
		},
	})
}

func TestAccFeatureFlag_IncludeInSnippetRevertToDefault(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.sdk_settings"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
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
				),
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
				),
			},
			// Clear specific value, check for default
			{
				Config: withRandomProjectIncludeInSnippetTrue(projectKey, testAccFeatureFlagIncludeInSnippetEmpty),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, KEY, "basic-flag-sdk-settings"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, INCLUDE_IN_SNIPPET, "true"),
				),
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
		client := testAccProvider.Meta().(*Client)
		_, _, err := client.ld.FeatureFlagsApi.GetFeatureFlag(client.ctx, projKey, flagKey).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting feature flag. %s", err)
		}
		return nil
	}
}
