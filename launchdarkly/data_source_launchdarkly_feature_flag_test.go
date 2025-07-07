package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go/v17"
	"github.com/stretchr/testify/require"
)

const (
	testAccDataSourceFeatureFlag = `
data "launchdarkly_feature_flag" "test" {
	key = "%s"
	project_key = "%s"
}	
`
)

func TestAccDataSourceFeatureFlag_noMatchReturnsError(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S)
	require.NoError(t, err)
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectBody := ldapi.ProjectPost{
		Name: "Terraform Flag Test Project",
		Key:  projectKey,
	}
	project, err := testAccProjectScaffoldCreate(client, projectBody)
	require.NoError(t, err)

	defer func() {
		err := testAccProjectScaffoldDelete(client, projectKey)
		require.NoError(t, err)
	}()

	flagKey := "nonexistent-flag"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccDataSourceFeatureFlag, flagKey, project.Key),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`Error: failed to get flag "nonexistent-flag" of project "%s": 404 Not Found:`, projectKey)),
			},
		},
	})
}

func TestAccDataSourceFeatureFlag_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S)
	require.NoError(t, err)

	flagName := "Flag Data Source Test"
	flagKey := "flag-ds-test"
	flagBody := ldapi.FeatureFlagBody{
		Name: flagName,
		Key:  flagKey,
		Variations: []ldapi.Variation{
			{Value: intfPtr(true)},
			{Value: intfPtr(false)},
		},
		Description: ldapi.PtrString("a flag to test the terraform flag data source"),
		Temporary:   ldapi.PtrBool(true),
		ClientSideAvailability: &ldapi.ClientSideAvailabilityPost{
			UsingEnvironmentId: true,
			UsingMobileKey:     false,
		},
	}
	flag, err := testAccFeatureFlagScaffold(client, projectKey, flagBody)
	require.NoError(t, err)

	defer func() {
		err := testAccProjectScaffoldDelete(client, projectKey)
		require.NoError(t, err)
	}()

	resourceName := "data.launchdarkly_feature_flag.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceFeatureFlag, flagKey, projectKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, KEY),
					resource.TestCheckResourceAttrSet(resourceName, NAME),
					resource.TestCheckResourceAttrSet(resourceName, PROJECT_KEY),
					resource.TestCheckResourceAttr(resourceName, KEY, flag.Key),
					resource.TestCheckResourceAttr(resourceName, NAME, flag.Name),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, *flag.Description),
					resource.TestCheckResourceAttr(resourceName, TEMPORARY, "true"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.value", "true"),
					resource.TestCheckResourceAttr(resourceName, "variations.1.value", "false"),
					resource.TestCheckResourceAttr(resourceName, ID, projectKey+"/"+flag.Key),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_environment_id", "true"),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_mobile_key", "false"),
				),
			},
		},
	})
}

func TestAccDataSourceFeatureFlag_withViews(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	flagKey := "test-flag-views"

	testAccDataSourceFeatureFlagWithViews := `
resource "launchdarkly_view" "test1" {
	project_key = "%s"
	key         = "test-view-1"
	name        = "Test View 1"
}

resource "launchdarkly_view" "test2" {
	project_key = "%s"
	key         = "test-view-2"
	name        = "Test View 2"
}

resource "launchdarkly_view_links" "test1" {
	project_key = "%s"
	view_key    = launchdarkly_view.test1.key
	
	flags = ["%s"]
}

resource "launchdarkly_view_links" "test2" {
	project_key = "%s"
	view_key    = launchdarkly_view.test2.key
	
	flags = ["%s"]
}

data "launchdarkly_feature_flag" "test" {
	project_key = "%s"
	key         = "%s"
	depends_on  = [launchdarkly_view_links.test1, launchdarkly_view_links.test2]
}
`

	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S)
	require.NoError(t, err)

	projectBody := ldapi.ProjectPost{
		Name: "Terraform Flag Views Test Project",
		Key:  projectKey,
	}
	_, err = testAccProjectScaffoldCreate(client, projectBody)
	require.NoError(t, err)

	flagName := "Test Flag with Views"
	flagBody := ldapi.FeatureFlagBody{
		Name: flagName,
		Key:  flagKey,
		Variations: []ldapi.Variation{
			{Value: intfPtr(true)},
			{Value: intfPtr(false)},
		},
		Description: ldapi.PtrString("a flag to test views in the terraform flag data source"),
		Temporary:   ldapi.PtrBool(false),
	}
	flag, err := testAccFeatureFlagScaffold(client, projectKey, flagBody)
	require.NoError(t, err)

	defer func() {
		err := testAccProjectScaffoldDelete(client, projectKey)
		require.NoError(t, err)
	}()

	resourceName := "data.launchdarkly_feature_flag.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceFeatureFlagWithViews,
					projectKey, projectKey, projectKey, flagKey, projectKey, flagKey, projectKey, flagKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "project_key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "key", flag.Key),
					resource.TestCheckResourceAttr(resourceName, "name", flag.Name),
					resource.TestCheckResourceAttr(resourceName, "views.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "views.*", "test-view-1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "views.*", "test-view-2"),
				),
			},
		},
	})
}
