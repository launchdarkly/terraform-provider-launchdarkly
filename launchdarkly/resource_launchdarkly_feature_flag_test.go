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
resource "launchdarkly_project" "testProject" {
	name = "testProject"
	key = "test-project"
}

resource "launchdarkly_feature_flag" "basic-flag" {
	project_key = "${launchdarkly_project.testProject.key}"
	key = "basic-flag"
	name = "Basic feature flag"
}
`
	testAccFeatureFlagUpdate = `
resource "launchdarkly_project" "testProject" {
	name = "testProject"
	key = "test-project"
}

resource "launchdarkly_feature_flag" "basic-flag" {
	project_key = "${launchdarkly_project.testProject.key}"
	key = "basic-flag"
	name = "Less basic feature flag"
	description = "this is a boolean flag by default becausethe variations field is omitted"
	tags = ["update", "terraform"]
	include_in_snippet = true
	temporary = true
}
`
	// The email must be set with a random name using fmt.Sprintf for this test to work since LD does
	// not support creating members with the same email address more than once.
	testAccFeatureFlagWithMaintainer = `
resource "launchdarkly_team_member" "teamMember1" {
	email = "%s@example.com"
	first_name = "first"
	last_name = "last"
	role = "admin"
	custom_roles = []
}

resource "launchdarkly_project" "testProject" {
	name = "testProject"
	key = "test-project"
}

resource "launchdarkly_feature_flag" "maintained-flag" {
	project_key = "${launchdarkly_project.testProject.key}"
	key = "maintained-flag"
	name = "Maintained feature flag"
	maintainer_id = "${launchdarkly_team_member.teamMember1.id}"
}
`

	testAccFeatureFlagCreateMultivariate = `
resource "launchdarkly_project" "testProject" {
	name = "testProject"
	key = "test-project"
}

resource "launchdarkly_feature_flag" "multivariate-flag-1" {
	project_key = "${launchdarkly_project.testProject.key}"
	key = "multivariate-flag-1"
	name = "multivariate flag 1 name"
	description = "this is a multivariate flag because we explicitly define the variations"
	variations = [
    	{
      		name = "variation1"
      		description = "a description"
			value = "string1"
		},
    	{
      		value = "string2"
		},
    	{
      		value = "another option"
    	},
  	]
  	tags = [
    	"this",
    	"is",
    	"unordered"
  	]
  	custom_properties = [
    	{
      		key = "some.property"
      		name = "Some Property"
      		value = [
        		"value1",
        		"value2",
        		"value3"
			]
    	},
    	{
      	key = "some.property2"
      	name = "Some Property"
      	value = ["very special custom property"]
    	}
	]
}
`
)

func TestAccFeatureFlag_Basic(t *testing.T) {
	resourceName := "launchdarkly_feature_flag.basic-flag"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccFeatureFlagBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.testProject"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "Basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, "key", "basic-flag"),
					resource.TestCheckResourceAttr(resourceName, "project_key", "test-project"),
				),
			},
		},
	})
}

func TestAccFeatureFlag_Update(t *testing.T) {
	resourceName := "launchdarkly_feature_flag.basic-flag"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccFeatureFlagBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.testProject"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "Basic feature flag"),
					resource.TestCheckResourceAttr(resourceName, "key", "basic-flag"),
					resource.TestCheckResourceAttr(resourceName, "project_key", "test-project"),
				),
			},
			{
				Config: testAccFeatureFlagUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.testProject"),
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

func TestAccFeatureFlag_WithMaintainer(t *testing.T) {
	randomName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag.maintained-flag"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccFeatureFlagWithMaintainer, randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.testProject"),
					testAccCheckMemberExists("launchdarkly_team_member.teamMember1"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "Maintained feature flag"),
					resource.TestCheckResourceAttr(resourceName, "key", "maintained-flag"),
					resource.TestCheckResourceAttr(resourceName, "project_key", "test-project"),
					resource.TestCheckResourceAttrPair(resourceName, "maintainer_id", "launchdarkly_team_member.teamMember1", "id"),
				),
			},
		},
	})
}

func TestAccFeatureFlag_CreateMultivariate(t *testing.T) {
	resourceName := "launchdarkly_feature_flag.multivariate-flag-1"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccFeatureFlagCreateMultivariate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.testProject"),
					testAccCheckFeatureFlagExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "multivariate flag 1 name"),
					resource.TestCheckResourceAttr(resourceName, "key", "multivariate-flag-1"),
					resource.TestCheckResourceAttr(resourceName, "project_key", "test-project"),
					resource.TestCheckResourceAttr(resourceName, "description", "this is a multivariate flag because we explicitly define the variations"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "3"),
					resource.TestCheckResourceAttr(resourceName, testAccVariationKey("string1", "description"), "a description"),
					resource.TestCheckResourceAttr(resourceName, testAccVariationKey("string1", "name"), "variation1"),
					resource.TestCheckResourceAttr(resourceName, testAccVariationKey("string1", "value"), "string1"),
					resource.TestCheckResourceAttr(resourceName, testAccVariationKey("string2", "value"), "string2"),
					resource.TestCheckResourceAttr(resourceName, testAccVariationKey("another option", "value"), "another option"),
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

func testAccVariationKey(val string, subKey string) string {
	return fmt.Sprintf("variations.%d.%s", hashcode.String(val), subKey)
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
			return fmt.Errorf("received an error getting environment. %s", err)
		}
		return nil
	}
}
