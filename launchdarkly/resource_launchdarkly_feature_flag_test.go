package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

const (
	testAccFeatureFlagBasic = `
resource "launchdarkly_project" "test" {
	name = "testProject"
	key = "test-project"
}

resource "launchdarkly_feature_flag" "basic" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag"
	name = "Basic feature flag"
	variation_type = "boolean"
}
`
	testAccFeatureFlagUpdate = `
resource "launchdarkly_project" "test" {
	name = "testProject"
	key = "test-project"
}

resource "launchdarkly_feature_flag" "basic" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag"
	name = "Less basic feature flag"
	variation_type = "boolean"
	description = "this is a boolean flag by default becausethe variations field is omitted"
	tags = ["update", "terraform"]
	include_in_snippet = true
	temporary = true
}
`

	testAccFeatureFlagNumber = `
resource "launchdarkly_project" "test" {
	name = "testProject"
	key = "test-project"
}

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
resource "launchdarkly_project" "test" {
	name = "testProject"
	key = "test-project"
}

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

resource "launchdarkly_project" "test" {
	name = "testProject"
	key = "test-project"
}

resource "launchdarkly_feature_flag" "maintained" {
	project_key = launchdarkly_project.test.key
	key = "maintained-flag"
	name = "Maintained feature flag"
	variation_type = "boolean"
	maintainer_id = launchdarkly_team_member.test.id
}
`

	testAccFeatureFlagCreateMultivariate = `
resource "launchdarkly_project" "test" {
	name = "testProject"
	key = "test-project"
}

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

	testAccFeatureFlagUpdateMultivariate = `
resource "launchdarkly_project" "test" {
	name = "testProject"
	key = "test-project"
}

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
)

func TestAccFeatureFlag_Basic(t *testing.T) {
	resourceName := "launchdarkly_feature_flag.basic"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccFeatureFlagBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, name, "Basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, key, "basic-flag"),
					resource.TestCheckResourceAttr(resourceName, project_key, "test-project"),
					resource.TestCheckResourceAttr(resourceName, variation_type, "boolean"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.value", "true"),
					resource.TestCheckResourceAttr(resourceName, "variations.1.value", "false"),
				),
			},
		},
	})
}

func TestAccFeatureFlag_Update(t *testing.T) {
	resourceName := "launchdarkly_feature_flag.basic"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccFeatureFlagBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "Basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, "key", "basic-flag"),
					resource.TestCheckResourceAttr(resourceName, "project_key", "test-project"),
				),
			},
			{
				Config: testAccFeatureFlagUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "Less basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, "key", "basic-flag"),
					resource.TestCheckResourceAttr(resourceName, "project_key", "test-project"),
					resource.TestCheckResourceAttr(resourceName, "description", "this is a boolean flag by default becausethe variations field is omitted"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("update"), "update"),
					resource.TestCheckResourceAttr(resourceName, testAccTagKey("terraform"), "terraform"),
					resource.TestCheckResourceAttr(resourceName, "include_in_snippet", "true"),
					resource.TestCheckResourceAttr(resourceName, "temporary", "true"),
				),
			},
		},
	})
}

func TestAccFeatureFlag_Number(t *testing.T) {
	resourceName := "launchdarkly_feature_flag.number"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccFeatureFlagNumber,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, name, "Number feature flag"),
					resource.TestCheckResourceAttr(resourceName, key, "numeric-flag"),
					resource.TestCheckResourceAttr(resourceName, project_key, "test-project"),
					resource.TestCheckResourceAttr(resourceName, variation_type, "number"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.value", "12.5"),
					resource.TestCheckResourceAttr(resourceName, "variations.1.value", "0"),
				),
			},
		},
	})
}

func TestAccFeatureFlag_JSON(t *testing.T) {
	resourceName := "launchdarkly_feature_flag.json"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccFeatureFlagJson,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, name, "JSON feature flag"),
					resource.TestCheckResourceAttr(resourceName, key, "json-flag"),
					resource.TestCheckResourceAttr(resourceName, project_key, "test-project"),
					resource.TestCheckResourceAttr(resourceName, variation_type, "json"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.value", `{"foo":"bar"}`),
					resource.TestCheckResourceAttr(resourceName, "variations.1.value", `{"extra":{"nested":"json"},"foo":"baz"}`),
				),
			},
		},
	})
}

func TestAccFeatureFlag_WithMaintainer(t *testing.T) {
	randomName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.maintained"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccFeatureFlagWithMaintainer, randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckMemberExists("launchdarkly_team_member.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "Maintained feature flag"),
					resource.TestCheckResourceAttr(resourceName, "key", "maintained-flag"),
					resource.TestCheckResourceAttr(resourceName, "project_key", "test-project"),
					resource.TestCheckResourceAttrPair(resourceName, "maintainer_id", "launchdarkly_team_member.test", "id"),
				),
			},
		},
	})
}

func TestAccFeatureFlag_CreateMultivariate(t *testing.T) {
	resourceName := "launchdarkly_feature_flag.multivariate"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccFeatureFlagCreateMultivariate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "multivariate flag 1 name"),
					resource.TestCheckResourceAttr(resourceName, "key", "multivariate-flag-1"),
					resource.TestCheckResourceAttr(resourceName, "project_key", "test-project"),
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

func TestAccFeatureFlag_UpdateMultivariate(t *testing.T) {
	resourceName := "launchdarkly_feature_flag.multivariate"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccFeatureFlagCreateMultivariate,
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
				Config: testAccFeatureFlagUpdateMultivariate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "multivariate flag 1 name"),
					resource.TestCheckResourceAttr(resourceName, "key", "multivariate-flag-1"),
					resource.TestCheckResourceAttr(resourceName, "project_key", "test-project"),
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
				Config: testAccFeatureFlagCreateMultivariate,
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

func testAccCustomPropertyKey(key string, subKey string) string {
	return fmt.Sprintf("custom_properties.%d.%s", hashcode.String(key), subKey)
}

func testAccCheckFeatureFlagExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		flagKey, ok := rs.Primary.Attributes[key]
		if !ok {
			return fmt.Errorf("feature flag key not found: %s", resourceName)
		}
		projKey, ok := rs.Primary.Attributes[project_key]
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
