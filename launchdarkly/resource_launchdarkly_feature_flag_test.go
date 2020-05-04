package launchdarkly

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
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
	default_on_variation = "true"
	default_off_variation = "false"
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

	testAccFeatureFlagJson = `
resource "launchdarkly_feature_flag" "json" {
	project_key = launchdarkly_project.test.key
	key         = "json-flag"
	name        = "JSON feature flag"
  
	variation_type = "json"
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
  }
`

	// The email must be set with a random name using fmt.Sprintf for this test to work since LD does
	// not support creating members with the same email address more than once.
	testAccFeatureFlagWithMaintainer = `
resource "launchdarkly_team_member" "test" {
	email = "%s@example.com"
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

	//testAccFeatureFlagWasMaintained is used to test that feature flag maintainers can be unset
	testAccFeatureFlagWasMaintained = `
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
}
`

	testAccFeatureFlagDefaults = `
resource "launchdarkly_feature_flag" "defaults" {
	project_key = launchdarkly_project.test.key
	key = "defaults-flag"
	name = "Feature flag with defaults"
	variation_type = "boolean"
	default_on_variation = "true"
	default_off_variation = "false"
}
`
	testAccFeatureFlagDefaultsUpdate = `
resource "launchdarkly_feature_flag" "defaults" {
	project_key = launchdarkly_project.test.key
	key = "defaults-flag"
	name = "Feature flag with defaults"
	variation_type = "boolean"
	default_on_variation = "true"
	default_off_variation = "true"
}
`
	testAccFeatureFlagDefaultsMissingOffInvalid = `
resource "launchdarkly_feature_flag" "defaults" {
	project_key = launchdarkly_project.test.key
	key = "defaults-flag"
	name = "Feature flag with defaults"
	variation_type = "boolean"
	default_on_variation = "a"
	default_off_variation = "b"
}
`

	testAccFeatureFlagDefaultsMultivariate = `
resource "launchdarkly_feature_flag" "defaults-multivariate" {
	project_key = launchdarkly_project.test.key
	key = "defaults-multivariate-flag"
	name = "Multivariate fature flag with defaults"
	variation_type = "string"
	default_on_variation = "b"
	default_off_variation = "b"
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
	name = "Multivariate fature flag with defaults"
	variation_type = "string"
	default_on_variation = "c"
	default_off_variation = "c"
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
	default_on_variation = "c"
	default_off_variation = "c"
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
)

func withRandomProject(randomProject, resource string) string {
	return fmt.Sprintf(`
	resource "launchdarkly_project" "test" {
		name = "testProject"
		key = "%s"
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
					resource.TestCheckNoResourceAttr(resourceName, "maintainer_id"),
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
					resource.TestCheckResourceAttr(resourceName, "name", "Basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, "key", "basic-flag"),
					resource.TestCheckResourceAttr(resourceName, "project_key", projectKey),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "Less basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, "key", "basic-flag"),
					resource.TestCheckResourceAttr(resourceName, "project_key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "description", "this is a boolean flag by default becausethe variations field is omitted"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("update"), "update"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("terraform"), "terraform"),
					resource.TestCheckResourceAttr(resourceName, "include_in_snippet", "true"),
					resource.TestCheckResourceAttr(resourceName, "temporary", "true"),
					resource.TestCheckResourceAttr(resourceName, "default_on_variation", "true"),
					resource.TestCheckResourceAttr(resourceName, "default_off_variation", "false"),
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
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
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
					resource.TestCheckResourceAttr(resourceName, "variations.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.value", `{"foo":"bar"}`),
					resource.TestCheckResourceAttr(resourceName, "variations.1.value", `{"extra":{"nested":"json"},"foo":"baz"}`),
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
					resource.TestCheckResourceAttr(resourceName, "name", "Maintained feature flag"),
					resource.TestCheckResourceAttr(resourceName, "key", "maintained-flag"),
					resource.TestCheckResourceAttr(resourceName, "project_key", projectKey),
					resource.TestCheckResourceAttrPair(resourceName, "maintainer_id", "launchdarkly_team_member.test", "id"),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagWasMaintained),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "Maintained feature flag"),
					resource.TestCheckResourceAttr(resourceName, "key", "maintained-flag"),
					resource.TestCheckResourceAttr(resourceName, "project_key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "maintainer_id", ""),
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
					resource.TestCheckResourceAttr(resourceName, "name", "Maintained feature flag"),
					resource.TestCheckResourceAttr(resourceName, "key", "maintained-flag"),
					resource.TestCheckResourceAttr(resourceName, "project_key", projectKey),
					resource.TestCheckResourceAttrPair(resourceName, "maintainer_id", "launchdarkly_team_member.test", "id"),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagWasMaintained),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "Maintained feature flag"),
					resource.TestCheckResourceAttr(resourceName, "key", "maintained-flag"),
					resource.TestCheckResourceAttr(resourceName, "project_key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "maintainer_id", ""),
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
					resource.TestCheckResourceAttr(resourceName, "name", "multivariate flag 1 name"),
					resource.TestCheckResourceAttr(resourceName, "key", "multivariate-flag-1"),
					resource.TestCheckResourceAttr(resourceName, "project_key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "description", "this is a multivariate flag because we explicitly define the variations"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.description", "a description"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.name", "variation1"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.value", "string1"),
					resource.TestCheckResourceAttr(resourceName, "variations.1.value", "string2"),
					resource.TestCheckResourceAttr(resourceName, "variations.2.value", "another option"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "3"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("this"), "this"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("is"), "is"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("unordered"), "unordered"),
					resource.TestCheckResourceAttr(resourceName, "custom_properties.#", "2"),
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property", "key"), "some.property"),
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property", "name"), "Some Property"),
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property", "value.#"), "3"),
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property", "value.0"), "value1"),
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property", "value.1"), "value2"),
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property", "value.2"), "value3"),
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property2", "key"), "some.property2"),
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property2", "name"), "Some Property"),
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property2", "value.#"), "1"),
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property2", "value.0"), "very special custom property"),
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
					resource.TestCheckResourceAttr(resourceName, "name", "multivariate flag 2 name"),
					resource.TestCheckResourceAttr(resourceName, "key", "multivariate-flag-2"),
					resource.TestCheckResourceAttr(resourceName, "project_key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "description", "this is a multivariate flag to test big number values"),
					resource.TestCheckResourceAttr(resourceName, "variation_type", "number"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.description", "a description"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.name", "variation1"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.value", "86400000"),
					resource.TestCheckResourceAttr(resourceName, "variations.1.value", "123"),
					resource.TestCheckResourceAttr(resourceName, "variations.2.value", "123456789"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "3"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("this"), "this"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("is"), "is"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("unordered"), "unordered"),
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
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property", "key"), "some.property"),
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property", "name"), "Some Property"),
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property", "value.#"), "3"),
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property", "value.0"), "value1"),
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property", "value.1"), "value2"),
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property", "value.2"), "value3"),
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property2", "key"), "some.property2"),
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property2", "name"), "Some Property"),
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property2", "value.#"), "1"),
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property2", "value.0"), "very special custom property"),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagUpdateMultivariate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "multivariate flag 1 name"),
					resource.TestCheckResourceAttr(resourceName, "key", "multivariate-flag-1"),
					resource.TestCheckResourceAttr(resourceName, "project_key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "description", "this is a multivariate flag because we explicitly define the variations"),
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
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("this"), "this"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("is"), "is"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("unordered"), "unordered"),
					resource.TestCheckResourceAttr(resourceName, "custom_properties.#", "1"),
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property", "key"), "some.property"),
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property", "name"), "Some Property Updated"),
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property", "value.#"), "2"),
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property", "value.0"), "value1"),
					resource.TestCheckResourceAttr(resourceName, testAccCustomPropertyKey("some.property", "value.1"), "value3"),
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
				),
			},
		},
	})
}

func TestAcccFeatureFlag_DefaultsInvalid(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      withRandomProject(projectKey, testAccFeatureFlagDefaultsMissingOffInvalid),
				ExpectError: regexp.MustCompile(`invalid default variations: default_on_variation "a" is not defined as a variation`),
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
					resource.TestCheckResourceAttr(resourceName, "default_on_variation", "true"),
					resource.TestCheckResourceAttr(resourceName, "default_off_variation", "false"),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagDefaultsUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "default_on_variation", "true"),
					resource.TestCheckResourceAttr(resourceName, "default_off_variation", "true"),
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
					resource.TestCheckResourceAttr(resourceName, "default_on_variation", "b"),
					resource.TestCheckResourceAttr(resourceName, "default_off_variation", "b"),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagDefaultsMultivariateUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "default_on_variation", "c"),
					resource.TestCheckResourceAttr(resourceName, "default_off_variation", "c"),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagDefaultsMultivariateUpdateRemoveVariation),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "default_on_variation", "c"),
					resource.TestCheckResourceAttr(resourceName, "default_off_variation", "c"),
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

func testAccCustomPropertyKey(key string, subKey string) string {
	return fmt.Sprintf("custom_properties.%d.%s", hashcode.String(key), subKey)
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
		_, _, err := client.ld.FeatureFlagsApi.GetFeatureFlag(client.ctx, projKey, flagKey, nil)
		if err != nil {
			return fmt.Errorf("received an error getting feature flag. %s", err)
		}
		return nil
	}
}
