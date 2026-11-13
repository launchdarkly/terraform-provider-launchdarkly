package launchdarkly

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go/v25"
	"github.com/stretchr/testify/require"
)

const testAccDataSourceEnvironmentReleasePlan = `
data "launchdarkly_environment_release_plan" "testing" {
	project_key     = "%s"
	environment_key = "%s"
	key             = "%s"
}
`

// testAccDataSourceEnvironmentReleasePlanScaffold creates a project and a
// release plan in its default "test" environment via the API.
func testAccDataSourceEnvironmentReleasePlanScaffold(client *Client, projectKey string) (*environmentReleasePlanRep, error) {
	projectBody := ldapi.ProjectPost{
		Name: "Environment Release Plan Test Project",
		Key:  projectKey,
	}
	project, err := testAccProjectScaffoldCreate(client, projectBody)
	if err != nil {
		return nil, err
	}

	contextKind := "user"
	post := environmentReleasePlanPostRequest{
		Key:           "erp-ds-plan",
		Name:          "ERP DS Plan",
		ReleaseMethod: RELEASE_PLAN_METHOD_PROGRESSIVE,
		ProgressiveRelease: environmentReleasePlanProgressiveRelease{
			RandomizationUnit: contextKind,
			Phases: []environmentReleasePlanPhase{
				{
					Steps: []environmentReleasePlanStep{
						{Allocation: 25000, DurationMillis: 3600000},
						{Allocation: 75000, DurationMillis: 3600000},
					},
					Clauses: []ldapi.Clause{
						{ContextKind: &contextKind, Attribute: "kind", Op: "in", Values: []interface{}{contextKind}},
					},
				},
			},
		},
	}
	return createEnvironmentReleasePlan(context.Background(), client, project.Key, "test", post)
}

func TestAccDataSourceEnvironmentReleasePlan_noMatchReturnsError(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectBody := ldapi.ProjectPost{
		Name: "Terraform Environment Release Plan Test Project",
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
				Config:      fmt.Sprintf(testAccDataSourceEnvironmentReleasePlan, project.Key, "test", "nonexistent-plan"),
				ExpectError: regexp.MustCompile("Error: 404 Not Found"),
			},
		},
	})
}

func TestAccDataSourceEnvironmentReleasePlan_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)

	plan, err := testAccDataSourceEnvironmentReleasePlanScaffold(client, projectKey)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, testAccProjectScaffoldDelete(client, projectKey))
	}()

	resourceName := "data.launchdarkly_environment_release_plan.testing"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceEnvironmentReleasePlan, projectKey, "test", plan.Key),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, ID, projectKey+"/test/"+plan.Key),
					resource.TestCheckResourceAttr(resourceName, KEY, plan.Key),
					resource.TestCheckResourceAttr(resourceName, NAME, plan.Name),
					resource.TestCheckResourceAttr(resourceName, ENVIRONMENT_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, RELEASE_METHOD, RELEASE_PLAN_METHOD_PROGRESSIVE),
					resource.TestCheckResourceAttr(resourceName, VERSION, "1"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.randomization_unit", "user"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.phases.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.phases.0.steps.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.phases.0.steps.1.allocation", "75000"),
					resource.TestCheckResourceAttr(resourceName, "progressive_release.phases.0.clauses.0.attribute", "kind"),
				),
			},
		},
	})
}
