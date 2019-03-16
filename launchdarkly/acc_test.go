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

var (
	testAccProviders = map[string]terraform.ResourceProvider{
		"launchdarkly": Provider().(*schema.Provider),
	}

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
	featureFlagBooleanCreate = projectCreateWithEnv + `
resource "launchdarkly_feature_flag" "boolean-flag-1" {
  project_key = "${launchdarkly_project.exampleproject2.key}"
  key = "boolean-flag-1"
  name = "boolean-flag-1 name"
  description = "this is a boolean flag by default because we omitted the variations field"
}
`

	featureFlagMultiVariateCreate = projectCreateWithEnv + `
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
    }]
}
`

	webhookCreate = `
resource "launchdarkly_webhook" "examplewebhook1" {
  name = "example-webhook"
  url = "http://webhooks.com"
  tags = [
    "terraform"
  ]
  secret = "THIS IS SUPER SECRET"
  sign = true,
  on = true,
}
`

	customRoleCreate = `
resource "launchdarkly_custom_role" "exampleCustomRole1" {
  key = "custom-role-key-1"
  name = "custom-role-name-1"
  description= "crd"
  policy = [{
 actions = ["*"]
effect = "allow"
resources = ["proj/*:env/production"]

}]
}
`

	teamMemberCreate = `
resource "launchdarkly_team_member" "teamMember2" {
    email = "member.%d@example.com"
    first_name = "first"
    last_name = "last"
    role = "admin"
    custom_roles = []
  }

`

	segmentCreate = `
resource "launchdarkly_segment" "segment3" {
    key = "segmentKey1"
project_key = "dummy-project"
env_key = "test"
  	name = "segment name"
description = "segment description"
tags = ["segmentTag1", "segmentTag2"]
}

`
)

func TestAccProjectCreateWithEnv(t *testing.T) {
	//projectKey := "accTestProject"

	resource.Test(t, resource.TestCase{
		IsUnitTest:                false,
		PreCheck:                  func() { checkCredentialsEnvVar(t) },
		Providers:                 testAccProviders,
		ProviderFactories:         nil,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              nil,
		Steps: []resource.TestStep{
			{
				Config: projectCreateWithEnv,
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
		IDRefreshName:   "",
		IDRefreshIgnore: nil,
	})
}

func TestFeatureFlagMultiVariateAcc(t *testing.T) {
	//projectKey := "accTestProject"

	resource.Test(t, resource.TestCase{
		IsUnitTest:                false,
		PreCheck:                  func() { checkCredentialsEnvVar(t) },
		Providers:                 testAccProviders,
		ProviderFactories:         nil,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              nil,
		Steps: []resource.TestStep{
			{
				Config: featureFlagMultiVariateCreate,
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
		IDRefreshName:   "",
		IDRefreshIgnore: nil,
	})
}

func TestFeatureFlagBooleanAcc(t *testing.T) {
	//projectKey := "accTestProject"

	resource.Test(t, resource.TestCase{
		IsUnitTest:                false,
		PreCheck:                  func() { checkCredentialsEnvVar(t) },
		Providers:                 testAccProviders,
		ProviderFactories:         nil,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              nil,
		Steps: []resource.TestStep{
			{
				Config: featureFlagBooleanCreate,
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
		IDRefreshName:   "",
		IDRefreshIgnore: nil,
	})
}
func TestWebhookAccExample(t *testing.T) {
	//projectKey := "accTestProject"

	resource.Test(t, resource.TestCase{
		IsUnitTest:                false,
		PreCheck:                  func() { checkCredentialsEnvVar(t) },
		Providers:                 testAccProviders,
		ProviderFactories:         nil,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              nil,
		Steps: []resource.TestStep{
			{
				Config: webhookCreate,
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
		IDRefreshName:   "",
		IDRefreshIgnore: nil,
	})
}

func TestCustomRoleAccExample(t *testing.T) {
	//projectKey := "accTestProject"

	resource.Test(t, resource.TestCase{
		IsUnitTest: false,
		PreCheck: func() {
			checkCredentialsEnvVar(t)
		},
		Providers:                 testAccProviders,
		ProviderFactories:         nil,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              nil,
		Steps: []resource.TestStep{
			{
				Config: customRoleCreate,
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
		IDRefreshName:   "",
		IDRefreshIgnore: nil,
	})
}

func TestTeamMemberAcc(t *testing.T) {
	//projectKey := "accTestProject"

	resource.Test(t, resource.TestCase{
		IsUnitTest: false,
		PreCheck: func() {
			checkCredentialsEnvVar(t)
		},
		Providers:                 testAccProviders,
		ProviderFactories:         nil,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              nil,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(teamMemberCreate, time.Now().Nanosecond()),
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
		IDRefreshName:   "",
		IDRefreshIgnore: nil,
	})
}
func TestSegmentAcc(t *testing.T) {
	//projectKey := "accTestProject"

	resource.Test(t, resource.TestCase{
		IsUnitTest: false,
		PreCheck: func() {
			checkCredentialsEnvVar(t)
		},
		Providers:                 testAccProviders,
		ProviderFactories:         nil,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              nil,
		Steps: []resource.TestStep{
			{
				Config: segmentCreate,
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
	err := cleanAccount(nil)
	if err != nil {
		t.Error(err)
	}
}

func cleanAccount(unused *terraform.State) error {
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
