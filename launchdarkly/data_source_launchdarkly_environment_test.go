package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go"
	"github.com/stretchr/testify/require"
)

const (
	testAccDataSourceEnvironment = `
data "launchdarkly_environment" "test" {
	key = "%s"
	project_key = "%s"
}
`
)

// testAccDataSourceEnvironmentScaffold creates a project with the given projectKey with the given env params
// for environment data source tests
func testAccDataSourceEnvironmentScaffold(client *Client, projectKey string, envBody ldapi.EnvironmentPost) (*ldapi.Environment, error) {
	// create project
	projectBody := ldapi.ProjectBody{
		Name: "Env Test Project",
		Key:  projectKey,
		Environments: []ldapi.EnvironmentPost{
			envBody,
		},
	}
	project, err := testAccDataSourceProjectCreate(client, projectBody)
	if err != nil {
		return nil, err
	}
	for _, env := range project.Environments {
		if env.Key == envBody.Key {
			return &env, nil
		}
	}
	return nil, fmt.Errorf("failed to create env")
}

func TestAccDataSourceEnvironment_noMatchReturnsError(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false)
	require.NoError(t, err)
	projectKey := "tf-env-test-proj"
	projectBody := ldapi.ProjectBody{
		Name: "Terraform Env Test Project",
		Key:  projectKey,
	}
	project, err := testAccDataSourceProjectCreate(client, projectBody)
	require.NoError(t, err)

	envKey := "bad-env-key"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccDataSourceEnvironment, envKey, project.Key),
				ExpectError: regexp.MustCompile(`errors during refresh: failed to get environment with key "bad-env-key" for project key: "tf-env-test-proj": 404 Not Found: {"message":"Unknown environment key bad-env-key"}`),
			},
		},
	})
	err = testAccDataSourceProjectDelete(client, projectKey)
	require.NoError(t, err)
}

func TestAccDataSourceEnv_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := "env-test-project"
	envName := "Terraform Test Env"
	envKey := "tf-test-env"
	envColor := "fff000"
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false)
	require.NoError(t, err)

	envBody := ldapi.EnvironmentPost{
		Name:       envName,
		Key:        envKey,
		Color:      envColor,
		SecureMode: true,
		Tags: []string{
			"some", "tag",
		},
	}

	env, err := testAccDataSourceEnvironmentScaffold(client, projectKey, envBody)
	require.NoError(t, err)

	resourceName := "data.launchdarkly_environment.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceEnvironment, envKey, projectKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "key"),
					resource.TestCheckResourceAttrSet(resourceName, "name"),
					resource.TestCheckResourceAttrSet(resourceName, "color"),
					resource.TestCheckResourceAttr(resourceName, "key", env.Key),
					resource.TestCheckResourceAttr(resourceName, "name", env.Name),
					resource.TestCheckResourceAttr(resourceName, "color", env.Color),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "mobile_key", env.MobileKey),
					resource.TestCheckResourceAttr(resourceName, "default_ttl", "0"),
					resource.TestCheckResourceAttr(resourceName, "id", projectKey+"/"+env.Key),
				),
			},
		},
	})

	err = testAccDataSourceProjectDelete(client, projectKey)
	require.NoError(t, err)
}
