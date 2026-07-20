package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccDataSourceAIAgentGraphMissing = `
data "launchdarkly_ai_agent_graph" "test" {
	project_key = "%s"
	key         = "%s"
}
`

func TestAccDataSourceAIAgentGraph_noMatchReturnsError(t *testing.T) {
	projectKey := "nonexistent-project-key"
	graphKey := "nonexistent-graph-key"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccDataSourceAIAgentGraphMissing, projectKey, graphKey),
				ExpectError: regexp.MustCompile(`failed to get agent graph with key "nonexistent-graph-key" in project "nonexistent-project-key"`),
			},
		},
	})
}

func TestAccDataSourceAIAgentGraph_exists(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	graphKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	dataSourceName := "data.launchdarkly_ai_agent_graph.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAIAgentGraphDestroy,
		Steps: []resource.TestStep{
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(`
resource "launchdarkly_ai_agent_graph" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "Data source test graph"
	description = "Graph for data source test"
}

data "launchdarkly_ai_agent_graph" "test" {
	project_key = launchdarkly_project.test.key
	key         = launchdarkly_ai_agent_graph.test.key
}
`, graphKey)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(dataSourceName, KEY, graphKey),
					resource.TestCheckResourceAttr(dataSourceName, NAME, "Data source test graph"),
					resource.TestCheckResourceAttr(dataSourceName, DESCRIPTION, "Graph for data source test"),
					resource.TestCheckResourceAttrSet(dataSourceName, CREATION_DATE),
				),
			},
		},
	})
}
