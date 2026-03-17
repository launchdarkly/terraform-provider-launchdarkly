package launchdarkly

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go/v22"
	"github.com/stretchr/testify/require"
)

func TestAccDataSourceFlagDefaults_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)

	projectBody := ldapi.ProjectPost{
		Name: "Flag Defaults DS Test",
		Key:  projectKey,
	}
	_, err = testAccProjectScaffoldCreate(client, projectBody)
	require.NoError(t, err)
	defer func() {
		err := testAccProjectScaffoldDelete(client, projectKey)
		require.NoError(t, err)
	}()

	resourceName := "data.launchdarkly_flag_defaults.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "launchdarkly_flag_defaults" "test" {
	project_key = "%s"
}
`, projectKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttrSet(resourceName, TEMPORARY),
					resource.TestCheckResourceAttr(resourceName, "boolean_defaults.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "boolean_defaults.0.on_variation"),
					resource.TestCheckResourceAttrSet(resourceName, "boolean_defaults.0.off_variation"),
				),
			},
		},
	})
}
