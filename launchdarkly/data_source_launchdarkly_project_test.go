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
	testAccProjectBasic = `
data "launchdarkly_project" "test" {
	key = "%s"
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
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccProjectBasic, projectKey),
				ExpectError: regexp.MustCompile(`Error: failed to get project with key "nonexistent-project-key": 404 Not Found`),
			},
		},
	})
}

func TestAccDataSourceProject_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectName := "Terraform Test Project"
	envName := "Test Environment"
	envKey := "test-environment"
	envColor := "000000"
	tag := "test-tag"
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)

	projectBody := ldapi.ProjectPost{
		Name: projectName,
		Key:  projectKey,
		DefaultClientSideAvailability: &ldapi.DefaultClientSideAvailabilityPost{
			UsingEnvironmentId: false,
			UsingMobileKey:     false,
		},
		Tags: []string{
			tag,
		},
		Environments: []ldapi.EnvironmentPost{
			{
				Name:            envName,
				Key:             envKey,
				Color:           envColor,
				SecureMode:      ldapi.PtrBool(true),
				ConfirmChanges:  ldapi.PtrBool(true),
				RequireComments: ldapi.PtrBool(true),
				Tags: []string{
					tag,
				},
			},
		},
	}

	project, err := testAccProjectScaffoldCreate(client, projectBody)
	require.NoError(t, err)

	defer func() {
		err := testAccProjectScaffoldDelete(client, projectKey)
		require.NoError(t, err)
	}()

	resourceName := "data.launchdarkly_project.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccProjectExists, projectKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, KEY),
					resource.TestCheckResourceAttrSet(resourceName, NAME),
					resource.TestCheckResourceAttr(resourceName, KEY, project.Key),
					resource.TestCheckResourceAttr(resourceName, NAME, project.Name),
					resource.TestCheckResourceAttr(resourceName, ID, project.Id),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
					// TODO: remove deprecated client_side_availability attribute tests pending next major release
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_environment_id", "false"),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.0.using_mobile_key", "false"),
					resource.TestCheckResourceAttr(resourceName, "default_client_side_availability.0.using_environment_id", "false"),
					resource.TestCheckResourceAttr(resourceName, "default_client_side_availability.0.using_mobile_key", "false"),
				),
			},
		},
	})
}
