package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go"
	"github.com/stretchr/testify/require"
)

const (
	testAccProjectBasic = `
data "launchdarkly_project" "test" {
	key = "%s"
	name = "%s"
	tags = [ "terraform", "test" ]
}
`

	testAccProjectExists = `
data "launchdarkly_project" "test" {
		key = "%s"
	}
	`
)

func TestAccDataSourceProject_noMatchReturnsError(t *testing.T) {
	projectKey := "nonexistent-project-key"
	projectName := "Nonexistent Project"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccProjectBasic, projectKey, projectName),
				ExpectError: regexp.MustCompile(`errors during refresh: failed to get project with key "nonexistent-project-key": 404 Not Found`),
			},
		},
	})
}

func TestAccDataSourceProject_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := "tf-test-project"
	projectName := "Terraform Test Project"
	envName := "Test Environment"
	envKey := "test-environment"
	envColor := "000000"
	tag := "test-tag"
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false)
	require.NoError(t, err)

	projectBody := ldapi.ProjectBody{
		Name:                      projectName,
		Key:                       projectKey,
		IncludeInSnippetByDefault: true,
		Tags: []string{
			tag,
		},
		Environments: []ldapi.EnvironmentPost{
			ldapi.EnvironmentPost{
				Name:       envName,
				Key:        envKey,
				Color:      envColor,
				SecureMode: true,
				Tags: []string{
					tag,
				},
			},
		},
	}

	project, err := testAccDataSourceProjectCreate(client, projectBody)
	require.NoError(t, err)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccProjectExists, projectKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.launchdarkly_project.test", "key"),
					resource.TestCheckResourceAttrSet("data.launchdarkly_project.test", "name"),
					resource.TestCheckResourceAttr("data.launchdarkly_project.test", "key", project.Key),
					resource.TestCheckResourceAttr("data.launchdarkly_project.test", "name", project.Name),
					resource.TestCheckResourceAttr("data.launchdarkly_project.test", "id", project.Id),
					resource.TestCheckResourceAttr("data.launchdarkly_project.test", "tags.#", "1"),
					resource.TestCheckResourceAttr("data.launchdarkly_project.test", "environments.0.key", project.Environments[0].Key),
					resource.TestCheckResourceAttr("data.launchdarkly_project.test", "environments.0.name", project.Environments[0].Name),
					resource.TestCheckResourceAttr("data.launchdarkly_project.test", "environments.0.color", project.Environments[0].Color),
				),
			},
		},
	})
	err = testAccDataSourceProjectDelete(client, projectKey)
	require.NoError(t, err)
}
