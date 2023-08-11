package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go/v12"
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
	projectBody := ldapi.ProjectPost{
		Name:         "Env Test Project",
		Key:          projectKey,
		Environments: []ldapi.EnvironmentPost{envBody},
	}
	project, err := testAccProjectScaffoldCreate(client, projectBody)
	if err != nil {
		return nil, err
	}
	for _, env := range project.Environments.Items {
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
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S)
	require.NoError(t, err)
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectBody := ldapi.ProjectPost{
		Name: "Terraform Env Test Project",
		Key:  projectKey,
	}
	project, err := testAccProjectScaffoldCreate(client, projectBody)
	require.NoError(t, err)

	defer func() {
		err := testAccProjectScaffoldDelete(client, projectKey)
		require.NoError(t, err)
	}()

	envKey := "bad-env-key"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccDataSourceEnvironment, envKey, project.Key),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`Error: failed to get environment with key "bad-env-key" for project key: "%s": 404 Not Found: {"message":"Unknown environment key bad-env-key"}`, projectKey)),
			},
		},
	})
}

func TestAccDataSourceEnv_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	envName := "Terraform Test Env"
	envKey := "tf-test-env"
	envColor := "fff000"
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S)
	require.NoError(t, err)

	envBody := ldapi.EnvironmentPost{
		Name:       envName,
		Key:        envKey,
		Color:      envColor,
		SecureMode: ldapi.PtrBool(true),
		Tags: []string{
			"some", "tag",
		},
	}

	env, err := testAccDataSourceEnvironmentScaffold(client, projectKey, envBody)
	require.NoError(t, err)

	defer func() {
		err := testAccProjectScaffoldDelete(client, projectKey)
		require.NoError(t, err)
	}()

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
					resource.TestCheckResourceAttrSet(resourceName, KEY),
					resource.TestCheckResourceAttrSet(resourceName, NAME),
					resource.TestCheckResourceAttrSet(resourceName, COLOR),
					resource.TestCheckResourceAttr(resourceName, KEY, env.Key),
					resource.TestCheckResourceAttr(resourceName, NAME, env.Name),
					resource.TestCheckResourceAttr(resourceName, COLOR, env.Color),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, MOBILE_KEY, env.MobileKey),
					resource.TestCheckResourceAttr(resourceName, DEFAULT_TTL, "0"),
					resource.TestCheckResourceAttr(resourceName, ID, projectKey+"/"+env.Key),
				),
			},
		},
	})
}
