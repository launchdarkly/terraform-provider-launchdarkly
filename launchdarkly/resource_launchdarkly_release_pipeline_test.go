package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// The release pipeline config references two environments that exist in the
// project from creation (`test` and `production`). The project is declared
// with both so the audiences resolve.
const testAccReleasePipelineProject = `
resource "launchdarkly_project" "test" {
	lifecycle {
		ignore_changes = [environments]
	}
	name = "Release Pipeline Test Project"
	key  = "%s"
	environments = [
		{
			name  = "Test"
			key   = "test"
			color = "000000"
		},
		{
			name  = "Production"
			key   = "production"
			color = "417505"
		},
	]
}
`

const testAccReleasePipelineCreate = `
resource "launchdarkly_release_pipeline" "test" {
	project_key = launchdarkly_project.test.key
	key         = "checkout-rollout"
	name        = "Checkout rollout"
	description = "Roll out checkout changes safely"

	phases = [
		{
			name = "Internal testing"
			audiences = [
				{
					environment_key = "test"
					name            = "QA"
				},
			]
		},
		{
			name = "General availability"
			audiences = [
				{
					environment_key = "production"
					name            = "Everyone"
					configuration = {
						release_strategy = "manual"
						require_approval = true
					}
				},
			]
		},
	]

	tags = ["terraform-managed"]

	depends_on = [launchdarkly_project.test]
}
`

const testAccReleasePipelineUpdate = `
resource "launchdarkly_release_pipeline" "test" {
	project_key = launchdarkly_project.test.key
	key         = "checkout-rollout"
	name        = "Checkout rollout updated"
	description = "Roll out checkout changes safely"

	phases = [
		{
			name = "Internal testing"
			audiences = [
				{
					environment_key = "test"
					name            = "QA"
				},
			]
		},
	]

	tags = ["terraform-managed", "checkout"]

	depends_on = [launchdarkly_project.test]
}
`

func TestAccReleasePipeline_CreateUpdate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_release_pipeline.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckReleasePipelineDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccReleasePipelineProject, projectKey) + testAccReleasePipelineCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckReleasePipelineExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Checkout rollout"),
					resource.TestCheckResourceAttr(resourceName, KEY, "checkout-rollout"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, "phases.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "phases.0.name", "Internal testing"),
					resource.TestCheckResourceAttr(resourceName, "phases.0.audiences.0.environment_key", "test"),
					resource.TestCheckResourceAttr(resourceName, "phases.1.audiences.0.configuration.release_strategy", "manual"),
					resource.TestCheckResourceAttr(resourceName, "phases.1.audiences.0.configuration.require_approval", "true"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccReleasePipelineProject, projectKey) + testAccReleasePipelineUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckReleasePipelineExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Checkout rollout updated"),
					resource.TestCheckResourceAttr(resourceName, "phases.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "phases.0.audiences.0.environment_key", "test"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckReleasePipelineExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		key, ok := rs.Primary.Attributes[KEY]
		if !ok {
			return fmt.Errorf("release pipeline key not found: %s", resourceName)
		}
		projKey, ok := rs.Primary.Attributes[PROJECT_KEY]
		if !ok {
			return fmt.Errorf("project key not found: %s", resourceName)
		}
		beta, err := newReleasePipelineBetaClient(mustTestAccClient())
		if err != nil {
			return err
		}
		_, _, err = beta.ld.ReleasePipelinesBetaApi.GetReleasePipelineByKey(beta.ctx, projKey, key).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting release pipeline: %s", err)
		}
		return nil
	}
}

func testAccCheckReleasePipelineDestroy(s *terraform.State) error {
	beta, err := newReleasePipelineBetaClient(mustTestAccClient())
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_release_pipeline" {
			continue
		}
		projKey := rs.Primary.Attributes[PROJECT_KEY]
		key := rs.Primary.Attributes[KEY]
		_, res, err := beta.ld.ReleasePipelinesBetaApi.GetReleasePipelineByKey(beta.ctx, projKey, key).Execute()
		if isStatusNotFound(res) {
			continue
		}
		if err != nil {
			return fmt.Errorf("unexpected error checking release pipeline %q destruction in project %q: %s", key, projKey, handleLdapiErr(err))
		}
		return fmt.Errorf("release pipeline %q still exists in project %q", key, projKey)
	}
	return nil
}
