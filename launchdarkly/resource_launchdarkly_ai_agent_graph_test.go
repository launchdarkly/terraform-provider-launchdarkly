package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	// Metadata-only graph (no root_config_key / edges) — the API explicitly
	// supports this shape.
	testAccAIAgentGraphCreate = `
resource "launchdarkly_ai_agent_graph" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "%s"
	description = "%s"
}
`

	testAccAIAgentGraphUpdate = `
resource "launchdarkly_ai_agent_graph" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "%s"
	description = "%s"
}
`

	// Graph with a root config and a single edge connecting two AI Configs.
	testAccAIAgentGraphWithEdges = `
resource "launchdarkly_ai_config" "root" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "Root agent config"
	mode        = "agent"
}

resource "launchdarkly_ai_config" "child" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "Child agent config"
	mode        = "agent"
	depends_on  = [launchdarkly_ai_config.root]
}

resource "launchdarkly_ai_agent_graph" "test" {
	project_key     = launchdarkly_project.test.key
	key             = "%s"
	name            = "Graph with edges"
	root_config_key = launchdarkly_ai_config.root.key
	edges = [{
		key           = "root-to-child"
		source_config = launchdarkly_ai_config.root.key
		target_config = launchdarkly_ai_config.child.key
		handoff       = jsonencode({ reason = "escalate" })
	}]
}
`
)

func TestAccAIAgentGraph_CreateAndUpdate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	graphKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_agent_graph.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAIAgentGraphDestroy,
		Steps: []resource.TestStep{
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIAgentGraphCreate, graphKey, "Initial graph", "Initial description")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIAgentGraphExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, KEY, graphKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "Initial graph"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "Initial description"),
					resource.TestCheckResourceAttrSet(resourceName, CREATION_DATE),
					resource.TestCheckResourceAttrSet(resourceName, LAST_MODIFIED),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIAgentGraphUpdate, graphKey, "Updated graph", "Updated description")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIAgentGraphExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Updated graph"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "Updated description"),
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

func TestAccAIAgentGraph_WithEdges(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	rootKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	childKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	graphKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_agent_graph.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAIAgentGraphDestroy,
		Steps: []resource.TestStep{
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIAgentGraphWithEdges, rootKey, childKey, graphKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIAgentGraphExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, graphKey),
					resource.TestCheckResourceAttr(resourceName, ROOT_CONFIG_KEY, rootKey),
					resource.TestCheckResourceAttr(resourceName, "edges.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "edges.0.key", "root-to-child"),
					resource.TestCheckResourceAttr(resourceName, "edges.0.source_config", rootKey),
					resource.TestCheckResourceAttr(resourceName, "edges.0.target_config", childKey),
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

// TestAccAIAgentGraph_ClearForcesReplace verifies that reverting a graph with a
// root config and edges back to a metadata-only graph forces a destroy+recreate.
// The merge-patch update API cannot express clearing these fields, so the
// resource uses RequiresReplace for that transition instead.
func TestAccAIAgentGraph_ClearForcesReplace(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	rootKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	childKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	graphKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_agent_graph.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAIAgentGraphDestroy,
		Steps: []resource.TestStep{
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIAgentGraphWithEdges, rootKey, childKey, graphKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIAgentGraphExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ROOT_CONFIG_KEY, rootKey),
					resource.TestCheckResourceAttr(resourceName, "edges.#", "1"),
				),
			},
			{
				// Drop root_config_key + edges -> metadata-only graph at the same
				// key. This must plan as a replacement, not an in-place update.
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIAgentGraphCreate, graphKey, "Now metadata only", "Reverted to metadata-only")),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIAgentGraphExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Now metadata only"),
					resource.TestCheckNoResourceAttr(resourceName, ROOT_CONFIG_KEY),
				),
			},
		},
	})
}

func testAccCheckAIAgentGraphExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("agent graph ID is not set")
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		graphKey := rs.Primary.Attributes[KEY]

		// Agent graph endpoints live on the beta API surface; use a beta client
		// paired with the per-request LD-API-Version, matching the resource. The
		// standard client already sends a default LD-API-Version, so reusing it
		// here fails with "lDAPIVersion is required" (no header) or duplicates it.
		beta, err := newAIAgentGraphBetaClient(mustTestAccClient())
		if err != nil {
			return fmt.Errorf("failed to build beta client: %s", err)
		}
		_, _, err = beta.ld.AIConfigsApi.GetAgentGraph(beta.ctx, projectKey, graphKey).LDAPIVersion(agentGraphBetaVersion).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting agent graph: %s", err)
		}
		return nil
	}
}

var testAccCheckAIAgentGraphDestroy = func(s *terraform.State) error {
	beta, err := newAIAgentGraphBetaClient(mustTestAccClient())
	if err != nil {
		return fmt.Errorf("failed to build beta client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_ai_agent_graph" {
			continue
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		graphKey := rs.Primary.Attributes[KEY]

		_, res, err := beta.ld.AIConfigsApi.GetAgentGraph(beta.ctx, projectKey, graphKey).LDAPIVersion(agentGraphBetaVersion).Execute()
		if isStatusNotFound(res) {
			continue
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("agent graph %s/%s still exists", projectKey, graphKey)
	}
	return nil
}
