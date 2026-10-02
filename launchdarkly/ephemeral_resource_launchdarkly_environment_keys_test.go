package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	ldapi "github.com/launchdarkly/api-client-go/v24"
	"github.com/stretchr/testify/require"
)

// testAccProtoV6ProviderFactoriesWithEcho adds the test-only echo provider
// (echo_provider_test.go), which copies its `data` configuration into the
// `data` attribute of an `echo` resource. Ephemeral values can't be read from
// state directly, so echoing them is how acceptance tests assert on them.
var testAccProtoV6ProviderFactoriesWithEcho = map[string]func() (tfprotov6.ProviderServer, error){
	"launchdarkly": providerserver.NewProtocol6WithError(NewPluginProvider("test")()),
	"echo":         providerserver.NewProtocol6WithError(newTestEchoProvider()),
}

const testAccEphemeralEnvironmentKeys = `
ephemeral "launchdarkly_environment_keys" "test" {
	project_key = "%s"
	env_key     = "%s"
}

provider "echo" {
	data = ephemeral.launchdarkly_environment_keys.test
}

resource "echo" "test" {}
`

func TestAccEnvironmentKeysEphemeral_basic(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	envKey := "ephemeral-env"
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)

	env, err := testAccDataSourceEnvironmentScaffold(client, projectKey, ldapi.EnvironmentPost{
		Name:  "Ephemeral Keys Env",
		Key:   envKey,
		Color: "fff000",
	})
	require.NoError(t, err)

	defer func() {
		require.NoError(t, testAccProjectScaffoldDelete(client, projectKey))
	}()

	resourceName := "echo.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesWithEcho,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccEphemeralEnvironmentKeys, projectKey, envKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "data."+PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, "data."+ENV_KEY, envKey),
					resource.TestCheckResourceAttr(resourceName, "data."+API_KEY, env.ApiKey),
					resource.TestCheckResourceAttr(resourceName, "data."+MOBILE_KEY, env.MobileKey),
					resource.TestCheckResourceAttr(resourceName, "data."+CLIENT_SIDE_ID, env.Id),
					// Opening the ephemeral resource must never rotate keys.
					testAccCheckEnvironmentKeysUnchangedInLD(projectKey, envKey, &env.ApiKey, &env.MobileKey),
				),
			},
		},
	})
}

func TestAccEnvironmentKeysEphemeral_notFound(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)
	_, err = testAccProjectScaffoldCreate(client, ldapi.ProjectPost{Name: "Ephemeral Keys Not Found", Key: projectKey})
	require.NoError(t, err)

	defer func() {
		require.NoError(t, testAccProjectScaffoldDelete(client, projectKey))
	}()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesWithEcho,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccEphemeralEnvironmentKeys, projectKey, "bad-env-key"),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`failed to get environment with key "bad-env-key" for project key: "%s"`, projectKey)),
			},
		},
	})
}
