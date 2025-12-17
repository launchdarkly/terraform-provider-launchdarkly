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
	testAccDataSourceAIConfig = `
resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "test-ai-config"
	name        = "Test AI Config"
	description = "Test description"
	tags        = ["terraform", "test"]
}

data "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_ai_config.test.project_key
	key         = launchdarkly_ai_config.test.key
}
`
)

func TestAccDataSourceAIConfig_exists(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "data.launchdarkly_ai_config.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccDataSourceAIConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, ID),
					resource.TestCheckResourceAttr(resourceName, NAME, "Test AI Config"),
					resource.TestCheckResourceAttr(resourceName, KEY, "test-ai-config"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "Test description"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttrSet(resourceName, VERSION),
				),
			},
		},
	})
}

func TestAccDataSourceAIConfig_noMatchReturnsError(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	aiConfigKey := "nonexistent-ai-config-key"
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)

	projectBody := ldapi.ProjectPost{
		Name: "AI Config DS No Match Test",
		Key:  projectKey,
	}
	_, err = testAccProjectScaffoldCreate(client, projectBody)
	require.NoError(t, err)

	defer func() {
		err := testAccProjectScaffoldDelete(client, projectKey)
		require.NoError(t, err)
	}()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "launchdarkly_ai_config" "test" {
	project_key = "%s"
	key         = "%s"
}
`, projectKey, aiConfigKey),
				ExpectError: regexp.MustCompile(`failed to get AI config`),
			},
		},
	})
}
