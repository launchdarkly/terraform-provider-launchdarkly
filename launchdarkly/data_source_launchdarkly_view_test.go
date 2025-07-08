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
				ExpectError: regexp.MustCompile(`Project not found`),
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
	betaClient, err := newBetaClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S)
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

	view, err := createView(betaClient, projectKey, viewBody)
	require.NoError(t, err)

	defer func() {
		err := deleteView(betaClient, projectKey, viewKey)
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

func TestAccDataSourceView_withLinkedFlags(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectName := "view-discovery-test-" + projectKey
	resourceName := "data.launchdarkly_view.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "launchdarkly_project" "test" {
	name = "%s"
	key  = "%s"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_view" "test" {
	project_key = launchdarkly_project.test.key
	key         = "test-view"
	name        = "Test View"
	description = "Test view for discovery testing"
}

resource "launchdarkly_feature_flag" "test1" {
	project_key = launchdarkly_project.test.key
	key         = "test-flag-1"
	name        = "Test Flag 1"
	variation_type = "boolean"
}

resource "launchdarkly_feature_flag" "test2" {
	project_key = launchdarkly_project.test.key
	key         = "test-flag-2"
	name        = "Test Flag 2"
	variation_type = "boolean"
}

resource "launchdarkly_view_links" "test" {
	project_key = launchdarkly_project.test.key
	view_key    = launchdarkly_view.test.key
	
	flags = [
		launchdarkly_feature_flag.test1.key,
		launchdarkly_feature_flag.test2.key
	]
}

data "launchdarkly_view" "test" {
	project_key = launchdarkly_project.test.key
	key         = launchdarkly_view.test.key
	depends_on  = [launchdarkly_view_links.test]
}
`, projectName, projectKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "project_key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "key", "test-view"),
					resource.TestCheckResourceAttr(resourceName, "name", "Test View"),
					resource.TestCheckResourceAttr(resourceName, "linked_flags.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "linked_flags.*", "test-flag-1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "linked_flags.*", "test-flag-2"),
				),
			},
		},
	})
}
