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
	testAccDataSourceAIModelConfig = `
data "launchdarkly_ai_model_config" "testing" {
	key         = "%s"
	project_key = "%s"
}
`
)

func testAccDataSourceAIModelConfigScaffold(client *Client, projectKey string, modelConfigBody ldapi.ModelConfigPost) (*ldapi.ModelConfig, error) {
	projectBody := ldapi.ProjectPost{
		Name: "AI Model Config Test Project",
		Key:  projectKey,
	}
	_, err := testAccProjectScaffoldCreate(client, projectBody)
	if err != nil {
		return nil, err
	}

	modelConfig, _, err := client.ldBeta.AIConfigsBetaApi.PostModelConfig(client.ctx, projectKey).LDAPIVersion("beta").ModelConfigPost(modelConfigBody).Execute()
	if err != nil {
		return nil, err
	}

	return modelConfig, nil
}

func TestAccDataSourceAIModelConfig_noMatchReturnsError(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectBody := ldapi.ProjectPost{
		Name: "Terraform AI Model Config Test Project",
		Key:  projectKey,
	}
	project, err := testAccProjectScaffoldCreate(client, projectBody)
	require.NoError(t, err)

	defer func() {
		err := testAccProjectScaffoldDelete(client, projectKey)
		require.NoError(t, err)
	}()

	modelConfigKey := "nonexistent-ai-model-config"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccDataSourceAIModelConfig, modelConfigKey, project.Key),
				ExpectError: regexp.MustCompile("Error: 404 Not Found"),
			},
		},
	})
}

func TestAccDataSourceAIModelConfig_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)

	modelConfigKey := "ai-model-ds-testing"
	modelConfigName := "AI Model Data Source Test"
	modelConfigId := "gpt-4-test"
	modelProvider := "openai"
	icon := "openai-icon"

	modelConfigBody := ldapi.ModelConfigPost{
		Key:      modelConfigKey,
		Name:     modelConfigName,
		Id:       modelConfigId,
		Provider: &modelProvider,
		Icon:     &icon,
		Tags:     []string{"test", "datasource"},
	}
	modelConfig, err := testAccDataSourceAIModelConfigScaffold(client, projectKey, modelConfigBody)
	require.NoError(t, err)

	defer func() {
		err := testAccProjectScaffoldDelete(client, projectKey)
		require.NoError(t, err)
	}()

	resourceName := "data.launchdarkly_ai_model_config.testing"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceAIModelConfig, modelConfigKey, projectKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, KEY),
					resource.TestCheckResourceAttrSet(resourceName, NAME),
					resource.TestCheckResourceAttrSet(resourceName, PROJECT_KEY),
					resource.TestCheckResourceAttr(resourceName, KEY, modelConfig.Key),
					resource.TestCheckResourceAttr(resourceName, NAME, modelConfig.Name),
					resource.TestCheckResourceAttr(resourceName, MODEL_ID, modelConfig.Id),
					resource.TestCheckResourceAttr(resourceName, MODEL_PROVIDER, *modelConfig.Provider),
					resource.TestCheckResourceAttr(resourceName, ICON, *modelConfig.Icon),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
				),
			},
		},
	})
}
