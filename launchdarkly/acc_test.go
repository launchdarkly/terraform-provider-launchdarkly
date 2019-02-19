package launchdarkly

// Based on https://www.terraform.io/docs/extend/testing/acceptance-tests/testcase.html

import (
	"context"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/launchdarkly/api-client-go"
	"os"
	"testing"
)

var (
	testAccProviders = map[string]terraform.ResourceProvider{
		"launchdarkly": Provider().(*schema.Provider),
	}

	projectCreate = `
resource "launchdarkly_project" "exampleproject1" {
  name = "example-project"
  key = "example-project"
  tags = [
    "terraform"]
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

func TestAccExample(t *testing.T) {
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
				Config: projectCreate,
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
	TestCleanAccount(t)
}

func cleanAccount(unused *terraform.State) error {
	ctx := context.WithValue(context.Background(), ldapi.ContextAPIKey, ldapi.APIKey{
		Key: os.Getenv(launchDarklyApiKeyEnvVar),
	})

	client := ldapi.NewAPIClient(ldapi.NewConfiguration())

	// make sure we have a dummy project
	_, response, err := client.ProjectsApi.GetProject(ctx, "dummy-project")

	if response.StatusCode == 404 {
		_, err = client.ProjectsApi.PostProject(ctx, ldapi.ProjectBody{Name: "dummy-project", Key: "dummy-project"})
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
