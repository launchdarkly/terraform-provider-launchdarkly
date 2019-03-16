package launchdarkly

// Based on https://www.terraform.io/docs/extend/testing/acceptance-tests/testcase.html
// These tests are intended to exercise as many code paths as possible and show examples for each resource.
// The general pattern for each resource:
// 1. Create the resource
// 2. Update the resource
// 3. Import resource (see testAcc function)

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

const (
	testProject = `
resource "launchdarkly_project" "testProject" {
	name = "testProject"
	key = "test-project"
}`
)

var (
	providers = map[string]terraform.ResourceProvider{
		"launchdarkly": Provider().(*schema.Provider),
	}
)

func TestAccProject(t *testing.T) {
	projectCreate := `
resource "launchdarkly_project" "exampleproject2" {
	key = "example-project2"
	name = "example project 2"
	tags = [ "terraform" ]
}`
	projectUpdate := `
resource "launchdarkly_project" "exampleproject2" {
	key = "example-project2"
	name = "example project 2"
	tags = []
}`

	testAcc(t, "launchdarkly_project.exampleproject2", projectCreate, projectUpdate)
}

func TestAccProjectWithEnvironments(t *testing.T) {
	projectWithEnvsCreate := `
resource "launchdarkly_project" "exampleproject1" {
  key = "example-project1"
  name = "example project 1"
  tags = [ "terraform", "terraform2" ]
  environments = [
    {
      name = "defined in project post 1"
      key = "projDefinedEnv1"
      color = "0000f0"
      default_ttl = 100.0
      secure_mode = true
      default_track_events = false
    },
	{
      name = "defined in project post 2"
      key = "projDefinedEnv2"
      color = "0000ff"
      secure_mode = false
      default_track_events = true
    }
  ]
}
`

	projectWithEnvsUpdate := `
resource "launchdarkly_project" "exampleproject1" {
  key = "example-project1"
  name = "example project 1 with updated name"
  tags = [ "terraform1" ]
  environments = [
	{
      key = "projDefinedEnv1"
      name = "defined in project post 1 (update)"
      color = "000000"
      default_ttl = 100.0
      secure_mode = false
      default_track_events = true
    },
	{
      key = "projDefinedEnv2"
      name = "defined in project post 2 (update)"
      color = "ffffff"
    }
  ]
}
`

	testAcc(t, "launchdarkly_project.exampleproject1", projectWithEnvsCreate, projectWithEnvsUpdate)
}

func TestAccEnvironment(t *testing.T) {
	envCreate := testProject + `
resource "launchdarkly_environment" "staging1" {
	name = "Staging1"
  	key = "staging1"
  	color = "ff00ff"
  	secure_mode = true
  	default_track_events = false
  	default_ttl = 100.0
  	project_key = "${launchdarkly_project.testProject.key}"
}`

	envUpdate := testProject + `
resource "launchdarkly_environment" "staging1" {
	name = "Staging1 (update)"
  	key = "staging1"
  	color = "ff00fa"
  	secure_mode = false
  	default_track_events = true
  	default_ttl = 500.0
  	project_key = "${launchdarkly_project.testProject.key}"
}`

	testAcc(t, "launchdarkly_environment.staging1", envCreate, envUpdate)
}

func TestAccFeatureFlagMultivariate(t *testing.T) {
	featureFlagCreate := testProject + `
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
}`

	featureFlagUpdate := testProject + `
resource "launchdarkly_project" "projForFlagTest" {
	name = "project for testing flags"
	key = "projForFlagTest"
}

resource "launchdarkly_feature_flag" "multivariate-flag-1" {
	project_key = "${launchdarkly_project.testProject.key}"
	key = "multivariate-flag-1"
	name = "multivariate flag 1 name (update)"
	description = "updated description"
	variations = [
    	{
      		name = "variation1"
      		description = "a description we updated"
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
    	"is"
  	]
  	custom_properties = [
    	{
      		key = "some.property"
      		name = "Some Property"
      		value = [
        		"value1",
        		"value2",
        		"value3",
				"now with this new updated value!!!!"
			]
    	},
    	{
      	key = "some.property2"
      	name = "Some Property"
      	value = ["very special updated property"]
    	}
	]
}`

	testAcc(t, "launchdarkly_feature_flag.multivariate-flag-1", featureFlagCreate, featureFlagUpdate)
}

func TestFeatureFlagDefaultBooleanVariationsAcc(t *testing.T) {
	featureFlagCreateBoolean := testProject + `
resource "launchdarkly_feature_flag" "boolean-flag-1" {
  	project_key = "${launchdarkly_project.testProject.key}"
  	key = "boolean-flag-1"
  	name = "boolean-flag-1 name"
  	description = "this is a boolean flag by default because we omitted the variations field"
}`

	featureFlagUpdateBoolean := testProject + `
resource "launchdarkly_feature_flag" "boolean-flag-1" {
  	project_key = "${launchdarkly_project.testProject.key}"
  	key = "boolean-flag-1"
  	name = "boolean-flag-1 name (updated)"
  	description = "updated description"
  	tags = ["new_tag_who_dis"]
}`

	testAcc(t, "launchdarkly_feature_flag.boolean-flag-1", featureFlagCreateBoolean, featureFlagUpdateBoolean)
}

func TestSegmentAcc(t *testing.T) {
	segmentCreate := testProject + `
resource "launchdarkly_segment" "segment3" {
    key = "segmentKey1"
	project_key = "dummy-project"
	env_key = "test"
  	name = "segment name"
	description = "segment description"
	tags = ["segmentTag1", "segmentTag2"]
	included = ["user1", "user2"]
	excluded = ["user3", "user4"]
}`
	segmentUpdate := testProject + `
resource "launchdarkly_segment" "segment3" {
    key = "segmentKey1"
	project_key = "dummy-project"
	env_key = "test"
  	name = "segment name"
	description = "segment description"
	tags = ["segmentTag1", "segmentTag2"]
	included = ["user1", "user2", "user3", "user4"]
	excluded = []
}`

	testAcc(t, "launchdarkly_segment.segment3", segmentCreate, segmentUpdate)
}

func TestWebhookAccExample(t *testing.T) {
	webhookCreate := `
resource "launchdarkly_webhook" "examplewebhook1" {
	name = "example-webhook"
	url = "http://webhooks.com"
	tags = [ "terraform" ]
	on = true
}`
	webhookUpdate := `
resource "launchdarkly_webhook" "examplewebhook1" {
	name = "example-webhook"
	url = "http://webhooks.com/updatedUrl"
	tags = [ "terraform" ]
	on = true
}`

	testAcc(t, "launchdarkly_webhook.examplewebhook1", webhookCreate, webhookUpdate)
}

func TestCustomRoleAcc(t *testing.T) {
	customRoleCreate := `
resource "launchdarkly_custom_role" "customRole1" {
	key = "custom-role-key-1"
	name = "custom-role-name-1"
	description= "crd"
	policy = [
	{
		actions = ["*"]	
		effect = "allow"
		resources = ["proj/*:env/production"]
	}
	]
}
`

	customRoleUpdate := `
resource "launchdarkly_custom_role" "customRole1" {
	key = "custom-role-key-1"
	name = "custom-role-name-1"
	description= "crd"
	policy = [
	{
		actions = ["*"]	
		effect = "deny"
		resources = ["proj/*:env/production"]
	}
	]
}
`
	testAcc(t, "launchdarkly_custom_role.customRole1", customRoleCreate, customRoleUpdate)
}

func TestTeamMemberAcc(t *testing.T) {
	nanos := time.Now().Nanosecond()

	teamMemberCreate := fmt.Sprintf(`
resource "launchdarkly_team_member" "teamMember1" {
    email = "member.%d@example.com"
    first_name = "first"
    last_name = "last"
    role = "admin"
    custom_roles = []
}`, nanos)

	teamMemberUpdate := fmt.Sprintf(`
resource "launchdarkly_team_member" "teamMember1" {
    email = "member.%d@example.com"
    first_name = "first"
    last_name = "last"
    role = "writer"
    custom_roles = []
}`, nanos)

	testAcc(t, "launchdarkly_team_member.teamMember1", teamMemberCreate, teamMemberUpdate)
}

func testAcc(t *testing.T, resourceName string, config ...string) {
	testCase := resource.TestCase{
		PreCheck: func() {
			checkCredentialsEnvVar(t)
		},
		Providers: providers,
	}

	for _, c := range config {
		testCase.Steps = append(testCase.Steps, resource.TestStep{Config: c})
	}

	testCase.Steps = append(testCase.Steps, resource.TestStep{
		ResourceName:      resourceName,
		ImportState:       true,
		ImportStateVerify: true,
	})

	resource.Test(t, testCase)
}

func checkCredentialsEnvVar(t *testing.T) {
	if v := os.Getenv(launchDarklyApiKeyEnvVar); v == "" {
		t.Errorf("%s env var must be set for acceptance tests", launchDarklyApiKeyEnvVar)
	}
}
