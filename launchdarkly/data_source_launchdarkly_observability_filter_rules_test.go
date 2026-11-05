package launchdarkly

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go/v25"
	"github.com/stretchr/testify/require"
)

func TestAccDataSourceObservabilityFilterRules_basic(t *testing.T) {
	testAccObservabilityPreCheck(t)
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	client := mustTestAccClient()

	// Scaffold the project and a rule through the API, not Terraform.
	_, _, err := client.ld.ProjectsApi.PostProject(client.ctx).ProjectPost(*ldapi.NewProjectPost("Observability filter data source test", projectKey)).Execute()
	require.NoError(t, handleLdapiErr(err))
	t.Cleanup(func() {
		_, _ = client.ld.ProjectsApi.DeleteProject(client.ctx, projectKey).Execute()
	})
	_, err = createObservabilityFilterRule(context.Background(), client, projectKey, observabilityFilterRulePost{
		Key: "drop-healthz", Name: "Drop health checks", ProductType: LOGS, Enabled: true,
		Query: "route=/healthz", Operation: "exclude", Rate: 1, Attributes: []string{},
	})
	require.NoError(t, err)

	addr := "data.launchdarkly_observability_filter_rules.logs"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccObservabilityPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "launchdarkly_observability_filter_rules" "logs" {
  project_key  = "` + projectKey + `"
  product_type = "logs"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "id", projectKey+"/logs"),
					resource.TestCheckResourceAttr(addr, "rules.#", "1"),
					resource.TestCheckResourceAttr(addr, "rules.0.key", "drop-healthz"),
					resource.TestCheckResourceAttr(addr, "rules.0.operation", "exclude"),
				),
			},
		},
	})
}
