package launchdarkly

// Based on https://www.terraform.io/docs/extend/testing/acceptance-tests/testcase.html

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/launchdarkly/api-client-go"
	"os"
	"testing"
	"time"
)

const (
	projectCreateWithEnv = `
resource "launchdarkly_project" "exampleproject2" {
  name = "example-project"
  key = "example-project2"
  tags = [ "terraform" ]
  environments = [
    {
      name = "defined in project post"
      key = "projDefinedEnv"
      color = "0000f0"
      default_ttl = 100.0
      secure_mode = true
      default_track_events = false
    }
  ]
}
`
)

func TestAccProjectCreateWithEnv(t *testing.T) {
	testAcc(t, projectCreateWithEnv)
}

func TestAccProjectCreateWithoutEnv(t *testing.T) {
	testAcc(t, `
resource "launchdarkly_project" "exampleproject3" {
	name = "example-project3"
	key = "example-project3"
	tags = [ "terraform" ]
}`)
}

func TestAccEnvironmentCreate(t *testing.T) {
	testAcc(t, `
resource "launchdarkly_project" "projForEnvTest" {
	name = "project for testing environment creation"
	key = "projForEnvTest"
}

resource "launchdarkly_environment" "staging1" {
	name = "Staging1"
  	key = "staging1"
  	color = "ff00ff"
  	secure_mode = true
  	default_track_events = false
  	default_ttl = 100.0
  	project_key = "${launchdarkly_project.projForEnvTest.key}"
}`)
}

func TestFeatureFlagMultiVariateAcc(t *testing.T) {
	testAcc(t, projectCreateWithEnv+`
resource "launchdarkly_feature_flag" "multivariate-flag-2" {
	project_key = "${launchdarkly_project.exampleproject2.key}"
	key = "multivariate-flag-2"
	name = "multivariate-flag-2 name"
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
}`)
}

func TestFeatureFlagDefaultBooleanVariationsAcc(t *testing.T) {
	testAcc(t, projectCreateWithEnv+`
resource "launchdarkly_feature_flag" "boolean-flag-1" {
  	project_key = "${launchdarkly_project.exampleproject2.key}"
  	key = "boolean-flag-1"
  	name = "boolean-flag-1 name"
  	description = "this is a boolean flag by default because we omitted the variations field"
}
`)
}
func TestWebhookAccExample(t *testing.T) {
	testAcc(t, `
resource "launchdarkly_webhook" "examplewebhook1" {
	name = "example-webhook"
	url = "http://webhooks.com"
	tags = [ "terraform" ]
  	secret = "THIS IS SUPER SECRET"
	sign = true
	on = true
}`)
}

func TestCustomRoleAccExample(t *testing.T) {
	testAcc(t, `
resource "launchdarkly_custom_role" "exampleCustomRole1" {
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
}`)
}

func TestTeamMemberAcc(t *testing.T) {
	testAcc(t, fmt.Sprintf(`
resource "launchdarkly_team_member" "teamMember2" {
    email = "member.%d@example.com"
    first_name = "first"
    last_name = "last"
    role = "admin"
    custom_roles = []
}`, time.Now().Nanosecond()))
}

func TestSegmentAcc(t *testing.T) {
	testAcc(t, `
resource "launchdarkly_segment" "segment3" {
    key = "segmentKey1"
	project_key = "dummy-project"
	env_key = "test"
  	name = "segment name"
	description = "segment description"
	tags = ["segmentTag1", "segmentTag2"]
}`)
}

func testAcc(t *testing.T, config string) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: false,
		PreCheck: func() {
			checkCredentialsEnvVar(t)
		},
		Providers: map[string]terraform.ResourceProvider{
			"launchdarkly": Provider().(*schema.Provider),
		},
		ProviderFactories:         nil,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              nil,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
		IDRefreshName:   "",
		IDRefreshIgnore: nil,
	})
}

func checkCredentialsEnvVar(t *testing.T) {
	if v := os.Getenv(launchDarklyApiKeyEnvVar); v == "" {
		t.Fatalf("%s env var must be set for acceptance tests", launchDarklyApiKeyEnvVar)
	}
	err := cleanAccount()
	if err != nil {
		t.Error(err)
	}
}

func cleanAccount() error {
	ctx := context.WithValue(context.Background(), ldapi.ContextAPIKey, ldapi.APIKey{
		Key: os.Getenv(launchDarklyApiKeyEnvVar),
	})

	client := ldapi.NewAPIClient(ldapi.NewConfiguration())

	// make sure we have a dummy project
	_, response, err := client.ProjectsApi.GetProject(ctx, "dummy-project")

	if response.StatusCode == 404 {
		_, _, err = client.ProjectsApi.PostProject(ctx, ldapi.ProjectBody{Name: "dummy-project", Key: "dummy-project"})
		if err != nil {
			return err
		}
	} else {
		if err != nil {
			return err
		}
	}
	projects, _, err := client.ProjectsApi.GetProjects(ctx)
	if err != nil {
		return err
	}

	// delete all but dummy project
	for _, p := range projects.Items {
		if p.Key != "dummy-project" {
			_, err := client.ProjectsApi.DeleteProject(ctx, p.Key)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
