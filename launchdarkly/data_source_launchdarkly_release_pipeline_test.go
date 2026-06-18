package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go/v22"
	"github.com/stretchr/testify/require"
)

const testAccDataSourceReleasePipeline = `
data "launchdarkly_release_pipeline" "testing" {
	key         = "%s"
	project_key = "%s"
}
`

// testAccDataSourceReleasePipelineScaffold creates a project (with its default
// `test` and `production` environments) and a release pipeline referencing
// them, all via the API. The release pipeline endpoints are beta, so the
// pipeline is created with the beta client.
func testAccDataSourceReleasePipelineScaffold(client *Client, beta *Client, projectKey string) (*ldapi.ReleasePipeline, error) {
	projectBody := ldapi.ProjectPost{
		Name: "Release Pipeline Test Project",
		Key:  projectKey,
	}
	if _, err := testAccProjectScaffoldCreate(client, projectBody); err != nil {
		return nil, err
	}

	input := ldapi.CreateReleasePipelineInput{
		Key:         "rp-ds-pipeline",
		Name:        "RP DS Pipeline",
		Description: ldapi.PtrString("a release pipeline to test the terraform data source"),
		Tags:        []string{"test"},
		Phases: []ldapi.CreatePhaseInput{
			{
				Name: "Internal testing",
				Audiences: []ldapi.AudiencePost{
					{EnvironmentKey: "test", Name: "QA"},
				},
			},
			{
				Name: "General availability",
				Audiences: []ldapi.AudiencePost{
					{
						EnvironmentKey: "production",
						Name:           "Everyone",
						Configuration: &ldapi.AudienceConfiguration{
							ReleaseStrategy: "manual",
							RequireApproval: true,
						},
					},
				},
			},
		},
	}
	pipeline, _, err := beta.ld.ReleasePipelinesBetaApi.PostReleasePipeline(beta.ctx, projectKey).CreateReleasePipelineInput(input).Execute()
	if err != nil {
		return nil, err
	}
	return pipeline, nil
}

func TestAccDataSourceReleasePipeline_noMatchReturnsError(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectBody := ldapi.ProjectPost{
		Name: "Terraform Release Pipeline Test Project",
		Key:  projectKey,
	}
	project, err := testAccProjectScaffoldCreate(client, projectBody)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, testAccProjectScaffoldDelete(client, projectKey))
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccDataSourceReleasePipeline, "nonexistent-pipeline", project.Key),
				ExpectError: regexp.MustCompile("Error: 404 Not Found"),
			},
		},
	})
}

func TestAccDataSourceReleasePipeline_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)
	beta, err := newReleasePipelineBetaClient(client)
	require.NoError(t, err)

	pipeline, err := testAccDataSourceReleasePipelineScaffold(client, beta, projectKey)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, testAccProjectScaffoldDelete(client, projectKey))
	}()

	resourceName := "data.launchdarkly_release_pipeline.testing"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceReleasePipeline, pipeline.Key, projectKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, KEY, pipeline.Key),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "RP DS Pipeline"),
					resource.TestCheckResourceAttr(resourceName, "phases.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "phases.0.audiences.0.environment_key", "test"),
					resource.TestCheckResourceAttr(resourceName, "phases.1.audiences.0.configuration.release_strategy", "manual"),
				),
			},
		},
	})
}
