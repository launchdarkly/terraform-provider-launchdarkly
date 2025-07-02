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
	testAccDataSourceViewBasic = `
data "launchdarkly_view" "test" {
	project_key = "%s"
	key         = "%s"
}
`

	testAccDataSourceViewExists = `
data "launchdarkly_view" "test" {
	project_key = "%s"
	key         = "%s"
}
`
)

func TestAccDataSourceView_noMatchReturnsError(t *testing.T) {
	projectKey := "nonexistent-project-key"
	viewKey := "nonexistent-view-key"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccDataSourceViewBasic, projectKey, viewKey),
				ExpectError: regexp.MustCompile(`Error: failed to get view with key "nonexistent-view-key" in project "nonexistent-project-key": 404 Not Found`),
			},
		},
	})
}

func TestAccDataSourceView_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	viewKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	viewName := "Terraform Test View"
	viewDescription := "Test view description"
	tag := "test-tag"

	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S)
	require.NoError(t, err)

	projectBody := ldapi.ProjectPost{
		Name: "Terraform Test Project",
		Key:  projectKey,
		DefaultClientSideAvailability: &ldapi.DefaultClientSideAvailabilityPost{
			UsingEnvironmentId: false,
			UsingMobileKey:     false,
		},
	}

	project, err := testAccProjectScaffoldCreate(client, projectBody)
	require.NoError(t, err)

	defer func() {
		err := testAccProjectScaffoldDelete(client, projectKey)
		require.NoError(t, err)
	}()

	viewBody := map[string]interface{}{
		"key":         viewKey,
		"name":        viewName,
		"description": viewDescription,
		"tags":        []string{tag},
	}

	view, err := createView(client, projectKey, viewBody)
	require.NoError(t, err)

	defer func() {
		err := deleteView(client, projectKey, viewKey)
		require.NoError(t, err)
	}()

	resourceName := "data.launchdarkly_view.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceViewExists, projectKey, viewKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, PROJECT_KEY),
					resource.TestCheckResourceAttrSet(resourceName, KEY),
					resource.TestCheckResourceAttrSet(resourceName, NAME),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, project.Key),
					resource.TestCheckResourceAttr(resourceName, KEY, view.Key),
					resource.TestCheckResourceAttr(resourceName, NAME, view.Name),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, viewDescription),
					resource.TestCheckResourceAttr(resourceName, ID, view.Id),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
					resource.TestCheckResourceAttr(resourceName, GENERATE_SDK_KEYS, "false"),
					resource.TestCheckResourceAttr(resourceName, ARCHIVED, "false"),
				),
			},
		},
	})
}
