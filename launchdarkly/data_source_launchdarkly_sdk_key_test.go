package launchdarkly

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go/v23"
	"github.com/stretchr/testify/require"
)

const (
	testAccDataSourceSdkKey = `
data "launchdarkly_sdk_key" "test" {
	project_key     = "%s"
	environment_key = "%s"
	key             = "%s"
}
`
)

func TestAccDataSourceSdkKey_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	// New LaunchDarkly projects are created with default "test" and
	// "production" environments.
	environmentKey := "test"
	sdkKeyKey := "ds-test-sdk-key"
	sdkKeyName := "Data source test SDK key"
	sdkKeyDescription := "SDK key to test the terraform data source"

	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)
	betaClient, err := newBetaClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)

	_, err = testAccProjectScaffoldCreate(client, ldapi.ProjectPost{Name: "SDK Key Data Source Test", Key: projectKey})
	require.NoError(t, err)

	defer func() {
		require.NoError(t, testAccProjectScaffoldDelete(client, projectKey))
	}()

	post := ldapi.NewSdkKeyPost(sdkKeyKey, sdkKeyName)
	post.SetKind("sdk")
	post.SetDescription(sdkKeyDescription)
	sdkKey, err := createSdkKey(betaClient, projectKey, environmentKey, *post)
	require.NoError(t, err)

	resourceName := "data.launchdarkly_sdk_key.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceSdkKey, projectKey, environmentKey, sdkKeyKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, KEY, sdkKey.Key),
					resource.TestCheckResourceAttr(resourceName, NAME, sdkKey.Name),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, sdkKeyDescription),
					resource.TestCheckResourceAttr(resourceName, KIND, "sdk"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENVIRONMENT_KEY, environmentKey),
					resource.TestCheckResourceAttr(resourceName, ID, projectKey+"/"+environmentKey+"/"+sdkKeyKey),
					resource.TestCheckResourceAttr(resourceName, VALUE, sdkKey.Value),
					resource.TestCheckResourceAttrSet(resourceName, VERSION),
				),
			},
		},
	})
}
