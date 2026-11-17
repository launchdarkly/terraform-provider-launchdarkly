package launchdarkly

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	ldapi "github.com/launchdarkly/api-client-go/v24"
	"github.com/stretchr/testify/require"
)

const testAccEphemeralSdkKey = `
ephemeral "launchdarkly_sdk_key" "test" {
	project_key     = "%s"
	environment_key = "%s"
	key             = "%s"
}

provider "echo" {
	data = ephemeral.launchdarkly_sdk_key.test
}

resource "echo" "test" {}
`

func TestAccSdkKey_Ephemeral(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	// New LaunchDarkly projects are created with default "test" and
	// "production" environments.
	environmentKey := "test"
	sdkKeyKey := "ephemeral-test-sdk-key"
	sdkKeyName := "Ephemeral test SDK key"
	sdkKeyDescription := "SDK key to test the terraform ephemeral resource"

	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)
	betaClient, err := newBetaClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)

	_, err = testAccProjectScaffoldCreate(client, ldapi.ProjectPost{Name: "SDK Key Ephemeral Test", Key: projectKey})
	require.NoError(t, err)

	defer func() {
		require.NoError(t, testAccProjectScaffoldDelete(client, projectKey))
	}()

	post := ldapi.NewSdkKeyPost(sdkKeyKey, sdkKeyName)
	post.SetKind("sdk")
	post.SetDescription(sdkKeyDescription)
	sdkKey, err := createSdkKey(betaClient, projectKey, environmentKey, *post)
	require.NoError(t, err)

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
				Config: fmt.Sprintf(testAccEphemeralSdkKey, projectKey, environmentKey, sdkKeyKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "data."+ID, projectKey+"/"+environmentKey+"/"+sdkKeyKey),
					resource.TestCheckResourceAttr(resourceName, "data."+KEY, sdkKey.Key),
					resource.TestCheckResourceAttr(resourceName, "data."+NAME, sdkKeyName),
					resource.TestCheckResourceAttr(resourceName, "data."+DESCRIPTION, sdkKeyDescription),
					resource.TestCheckResourceAttr(resourceName, "data."+KIND, "sdk"),
					resource.TestCheckResourceAttr(resourceName, "data."+VALUE, sdkKey.Value),
					resource.TestCheckResourceAttrSet(resourceName, "data."+VERSION),
				),
			},
		},
	})
}
