package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go/v23"
	"github.com/stretchr/testify/require"
)

const testAccDataSourceReleasePolicy = `
data "launchdarkly_release_policy" "testing" {
	key         = "%s"
	project_key = "%s"
}
`

// testAccDataSourceReleasePolicyScaffold creates a project and a progressive
// release policy via the API. The release-policies endpoints are beta, so the
// policy is created with the beta client and an explicit beta API version.
func testAccDataSourceReleasePolicyScaffold(client *Client, beta *Client, projectKey string) (*ldapi.ReleasePolicy, error) {
	projectBody := ldapi.ProjectPost{
		Name: "Release Policy Test Project",
		Key:  projectKey,
	}
	project, err := testAccProjectScaffoldCreate(client, projectBody)
	if err != nil {
		return nil, err
	}

	contextKind := "user"
	post := ldapi.PostReleasePolicyRequest{
		Key:           "rp-ds-policy",
		Name:          "RP DS Policy",
		ReleaseMethod: ldapi.PROGRESSIVE_RELEASE,
		Scope: &ldapi.ReleasePolicyScope{
			EnvironmentKeys: []string{"test"},
		},
		ProgressiveReleaseConfig: &ldapi.ProgressiveReleaseConfig{
			RolloutContextKindKey: &contextKind,
			Stages: []ldapi.ReleasePolicyStage{
				{Allocation: 25, DurationMillis: 3600000},
				{Allocation: 75, DurationMillis: 3600000},
			},
		},
	}
	policy, _, err := beta.ld.ReleasePoliciesBetaApi.PostReleasePolicy(beta.ctx, project.Key).
		LDAPIVersion(RELEASE_POLICY_BETA_VERSION).
		PostReleasePolicyRequest(post).
		Execute()
	if err != nil {
		return nil, err
	}
	return policy, nil
}

func TestAccDataSourceReleasePolicy_noMatchReturnsError(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectBody := ldapi.ProjectPost{
		Name: "Terraform Release Policy Test Project",
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
				Config:      fmt.Sprintf(testAccDataSourceReleasePolicy, "nonexistent-policy", project.Key),
				ExpectError: regexp.MustCompile("Error: 404 Not Found"),
			},
		},
	})
}

func TestAccDataSourceReleasePolicy_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)
	beta, err := newReleasePolicyBetaClient(client)
	require.NoError(t, err)

	policy, err := testAccDataSourceReleasePolicyScaffold(client, beta, projectKey)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, testAccProjectScaffoldDelete(client, projectKey))
	}()

	resourceName := "data.launchdarkly_release_policy.testing"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceReleasePolicy, policy.Key, projectKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, KEY),
					resource.TestCheckResourceAttrSet(resourceName, NAME),
					resource.TestCheckResourceAttr(resourceName, KEY, policy.Key),
					resource.TestCheckResourceAttr(resourceName, NAME, policy.Name),
					resource.TestCheckResourceAttr(resourceName, RELEASE_METHOD, "progressive-release"),
					resource.TestCheckResourceAttr(resourceName, ID, projectKey+"/"+policy.Key),
					resource.TestCheckResourceAttr(resourceName, "progressive_release_config.stages.#", "2"),
				),
			},
		},
	})
}
