package launchdarkly

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	testAccAIConfigVariationCreate = `
resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "Parent AI Config"
	description = "Parent config for variation tests"
	tags        = ["test"]
}

resource "launchdarkly_ai_config_variation" "test" {
	project_key = launchdarkly_project.test.key
	config_key  = launchdarkly_ai_config.test.key
	key         = "%s"
	name        = "%s"
	messages = [{
		role    = "system"
		content = "You are a helpful assistant."
	}]
}
`

	testAccAIConfigVariationUpdate = `
resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "Parent AI Config"
	description = "Parent config for variation tests"
	tags        = ["test"]
}

resource "launchdarkly_ai_config_variation" "test" {
	project_key = launchdarkly_project.test.key
	config_key  = launchdarkly_ai_config.test.key
	key         = "%s"
	name        = "%s"
	messages = [{
		role    = "system"
		content = "You are an expert assistant."
	}, {
		role    = "user"
		content = "Hello!"
	}]
}
`
	testAccAIConfigVariationWithModelConfigKey = `
resource "launchdarkly_model_config" "test" {
	project_key    = launchdarkly_project.test.key
	key            = "%s"
	name           = "Test Model"
	model_id       = "gpt-4"
	model_provider = "openai"
}

resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "Parent AI Config"
	description = "Parent config for variation tests"
	tags        = ["test"]
	depends_on  = [launchdarkly_model_config.test]
}

resource "launchdarkly_ai_config_variation" "test" {
	project_key      = launchdarkly_project.test.key
	config_key       = launchdarkly_ai_config.test.key
	key              = "%s"
	name             = "Variation with model config"
	model_config_key = launchdarkly_model_config.test.key
	messages = [{
		role    = "system"
		content = "You are a helpful assistant."
	}]
}
`

	testAccAIConfigVariationAgentMode = `
resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "Agent Mode Config"
	description = "Agent mode parent"
	mode        = "agent"
}

resource "launchdarkly_ai_config_variation" "test" {
	project_key = launchdarkly_project.test.key
	config_key  = launchdarkly_ai_config.test.key
	key         = "%s"
	name        = "%s"
}
`

	testAccAIConfigVariationWithInlineModel = `
resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "Parent AI Config"
	description = "Parent for inline model test"
	tags        = ["test"]
}

resource "launchdarkly_ai_config_variation" "test" {
	project_key = launchdarkly_project.test.key
	config_key  = launchdarkly_ai_config.test.key
	key         = "%s"
	name        = "Variation with inline model"
	model       = jsonencode({
		modelName  = "gpt-4"
		parameters = { temperature = 0.7 }
	})
	messages = [{
		role    = "system"
		content = "You are a helpful assistant."
	}]
}
`

	testAccAIConfigVariationWithJudges = `
resource "launchdarkly_ai_config" "quality_judge" {
	project_key           = launchdarkly_project.test.key
	key                   = "%[1]s"
	name                  = "Quality Judge"
	mode                  = "judge"
	evaluation_metric_key = "$ld:ai:judge:%[1]s"
}

resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%[2]s"
	name        = "Parent AI Config"
	description = "Parent for judges test"
	depends_on  = [launchdarkly_ai_config.quality_judge]
}

resource "launchdarkly_ai_config_variation" "test" {
	project_key = launchdarkly_project.test.key
	config_key  = launchdarkly_ai_config.test.key
	key         = "%[3]s"
	name        = "Variation with judges"
	messages = [{
		role    = "system"
		content = "You are a helpful assistant."
	}]
	judges = {
		(launchdarkly_ai_config.quality_judge.key) = {
			sampling_rate = 0.1
		}
	}
}
`

	testAccAIConfigVariationWithJudgesUpdate = `
resource "launchdarkly_ai_config" "quality_judge" {
	project_key           = launchdarkly_project.test.key
	key                   = "%[1]s"
	name                  = "Quality Judge"
	mode                  = "judge"
	evaluation_metric_key = "$ld:ai:judge:%[1]s"
}

resource "launchdarkly_ai_config" "accuracy_judge" {
	project_key           = launchdarkly_project.test.key
	key                   = "%[2]s"
	name                  = "Accuracy Judge"
	mode                  = "judge"
	evaluation_metric_key = "$ld:ai:judge:%[2]s"
	depends_on            = [launchdarkly_ai_config.quality_judge]
}

resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%[3]s"
	name        = "Parent AI Config"
	description = "Parent for judges test"
	depends_on  = [launchdarkly_ai_config.accuracy_judge]
}

resource "launchdarkly_ai_config_variation" "test" {
	project_key = launchdarkly_project.test.key
	config_key  = launchdarkly_ai_config.test.key
	key         = "%[4]s"
	name        = "Variation with judges"
	messages = [{
		role    = "system"
		content = "You are a helpful assistant."
	}]
	judges = {
		(launchdarkly_ai_config.quality_judge.key) = {
			sampling_rate = 0.25
		}
		(launchdarkly_ai_config.accuracy_judge.key) = {
			sampling_rate = 1
		}
	}
}
`

	testAccAIConfigVariationWithJudgesRemoved = `
resource "launchdarkly_ai_config" "quality_judge" {
	project_key           = launchdarkly_project.test.key
	key                   = "%[1]s"
	name                  = "Quality Judge"
	mode                  = "judge"
	evaluation_metric_key = "$ld:ai:judge:%[1]s"
}

resource "launchdarkly_ai_config" "accuracy_judge" {
	project_key           = launchdarkly_project.test.key
	key                   = "%[2]s"
	name                  = "Accuracy Judge"
	mode                  = "judge"
	evaluation_metric_key = "$ld:ai:judge:%[2]s"
	depends_on            = [launchdarkly_ai_config.quality_judge]
}

resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%[3]s"
	name        = "Parent AI Config"
	description = "Parent for judges test"
	depends_on  = [launchdarkly_ai_config.accuracy_judge]
}

resource "launchdarkly_ai_config_variation" "test" {
	project_key = launchdarkly_project.test.key
	config_key  = launchdarkly_ai_config.test.key
	key         = "%[4]s"
	name        = "Variation with judges"
	messages = [{
		role    = "system"
		content = "You are a helpful assistant."
	}]
}
`

	testAccAIConfigVariationWithToolKeys = `
resource "launchdarkly_ai_tool" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	description = "Test tool"
	schema_json = jsonencode({
		type = "object"
		properties = {
			query = { type = "string" }
		}
	})
}

resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "Parent AI Config"
	description = "Parent for tool keys test"
	depends_on  = [launchdarkly_ai_tool.test]
}

resource "launchdarkly_ai_config_variation" "test" {
	project_key = launchdarkly_project.test.key
	config_key  = launchdarkly_ai_config.test.key
	key         = "%s"
	name        = "Variation with tools"
	tool_keys   = [launchdarkly_ai_tool.test.key]
	messages = [{
		role    = "system"
		content = "You are a helpful assistant."
	}]
}
`

	testAccAIConfigVariationWithToolKeysUpdate = `
resource "launchdarkly_ai_tool" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%[1]s"
	description = "Test tool"
	schema_json = jsonencode({
		type = "object"
		properties = {
			query = { type = "string" }
		}
	})
}

resource "launchdarkly_ai_tool" "second" {
	project_key = launchdarkly_project.test.key
	key         = "%[2]s"
	description = "Second test tool"
	schema_json = jsonencode({
		type = "object"
		properties = {
			id = { type = "string" }
		}
	})
	depends_on = [launchdarkly_ai_tool.test]
}

resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%[3]s"
	name        = "Parent AI Config"
	description = "Parent for tool keys test"
	depends_on  = [launchdarkly_ai_tool.second]
}

resource "launchdarkly_ai_config_variation" "test" {
	project_key = launchdarkly_project.test.key
	config_key  = launchdarkly_ai_config.test.key
	key         = "%[4]s"
	name        = "Variation with tools"
	tool_keys   = [launchdarkly_ai_tool.test.key, launchdarkly_ai_tool.second.key]
	messages = [{
		role    = "system"
		content = "You are a helpful assistant."
	}]
}
`

	testAccAIConfigVariationWithToolKeysRemoved = `
resource "launchdarkly_ai_tool" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%[1]s"
	description = "Test tool"
	schema_json = jsonencode({
		type = "object"
		properties = {
			query = { type = "string" }
		}
	})
}

resource "launchdarkly_ai_tool" "second" {
	project_key = launchdarkly_project.test.key
	key         = "%[2]s"
	description = "Second test tool"
	schema_json = jsonencode({
		type = "object"
		properties = {
			id = { type = "string" }
		}
	})
	depends_on = [launchdarkly_ai_tool.test]
}

resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%[3]s"
	name        = "Parent AI Config"
	description = "Parent for tool keys test"
	depends_on  = [launchdarkly_ai_tool.second]
}

resource "launchdarkly_ai_config_variation" "test" {
	project_key = launchdarkly_project.test.key
	config_key  = launchdarkly_ai_config.test.key
	key         = "%[4]s"
	name        = "Variation with tools"
	tool_keys   = []
	messages = [{
		role    = "system"
		content = "You are a helpful assistant."
	}]
}
`
)

func TestAccAIConfigVariation_CreateAndUpdate(t *testing.T) {
	aiTestCooldown()
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	configKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	variationKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	variationName := "Test Variation"
	updatedVariationName := "Updated Variation"
	resourceName := "launchdarkly_ai_config_variation.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAIConfigVariationDestroy,
		Steps: []resource.TestStep{
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigVariationCreate, configKey, variationKey, variationName)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigVariationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, AI_CONFIG_KEY, configKey),
					resource.TestCheckResourceAttr(resourceName, KEY, variationKey),
					resource.TestCheckResourceAttr(resourceName, NAME, variationName),
					resource.TestCheckResourceAttr(resourceName, "messages.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "messages.0.role", "system"),
					resource.TestCheckResourceAttr(resourceName, "messages.0.content", "You are a helpful assistant."),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigVariationUpdate, configKey, variationKey, updatedVariationName)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigVariationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, updatedVariationName),
					resource.TestCheckResourceAttr(resourceName, "messages.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "messages.0.role", "system"),
					resource.TestCheckResourceAttr(resourceName, "messages.0.content", "You are an expert assistant."),
					resource.TestCheckResourceAttr(resourceName, "messages.1.role", "user"),
					resource.TestCheckResourceAttr(resourceName, "messages.1.content", "Hello!"),
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

func TestAccAIConfigVariation_WithModelConfigKey(t *testing.T) {
	aiTestCooldown()
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	configKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	modelConfigKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	variationKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_config_variation.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAIConfigVariationDestroy,
		Steps: []resource.TestStep{
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigVariationWithModelConfigKey, modelConfigKey, configKey, variationKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigVariationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, MODEL_CONFIG_KEY, modelConfigKey),
					resource.TestCheckResourceAttr(resourceName, "messages.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, VARIATION_ID),
					resource.TestCheckResourceAttrSet(resourceName, VERSION),
					resource.TestCheckResourceAttrSet(resourceName, CREATION_DATE),
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

// TestAccAIConfigVariation_AgentMode tests creating a variation under an agent-mode AI config.
// Note: description/instructions fields on variations are not yet reliably persisted by the
// API on POST, so this test only verifies basic creation and update under agent mode.
func TestAccAIConfigVariation_AgentMode(t *testing.T) {
	aiTestCooldown()
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	configKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	variationKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	variationName := "Agent Variation"
	updatedName := "Updated Agent Variation"
	resourceName := "launchdarkly_ai_config_variation.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAIConfigVariationDestroy,
		Steps: []resource.TestStep{
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigVariationAgentMode, configKey, variationKey, variationName)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigVariationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, variationName),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigVariationAgentMode, configKey, variationKey, updatedName)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigVariationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, updatedName),
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

// TestAccAIConfigVariation_WithJudges tests attaching judge AI configs to a
// variation: create with one judge, update to add a second and change a
// sampling rate, then remove all judges and confirm convergence on null.
func TestAccAIConfigVariation_WithJudges(t *testing.T) {
	aiTestCooldown()
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	qualityJudgeKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	accuracyJudgeKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	configKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	variationKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_config_variation.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAIConfigVariationDestroy,
		Steps: []resource.TestStep{
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigVariationWithJudges, qualityJudgeKey, configKey, variationKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigVariationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "judges.%", "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("judges.%s.sampling_rate", qualityJudgeKey), "0.1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigVariationWithJudgesUpdate, qualityJudgeKey, accuracyJudgeKey, configKey, variationKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigVariationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "judges.%", "2"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("judges.%s.sampling_rate", qualityJudgeKey), "0.25"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("judges.%s.sampling_rate", accuracyJudgeKey), "1"),
				),
			},
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigVariationWithJudgesRemoved, qualityJudgeKey, accuracyJudgeKey, configKey, variationKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigVariationExists(resourceName),
					resource.TestCheckNoResourceAttr(resourceName, "judges.%"),
				),
			},
		},
	})
}

// TestAccAIConfigVariation_WithToolKeys tests tool attachment end to end:
// create with one tool, update to two, then remove all with an explicit
// empty set. Every step asserts attachment through the API — state-only
// checks are masked by the read's preserve-prior fallback.
func TestAccAIConfigVariation_WithToolKeys(t *testing.T) {
	aiTestCooldown()
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	configKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	toolKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	secondToolKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	variationKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_config_variation.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAIConfigVariationDestroy,
		Steps: []resource.TestStep{
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigVariationWithToolKeys, toolKey, configKey, variationKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigVariationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Variation with tools"),
					resource.TestCheckResourceAttr(resourceName, "tool_keys.#", "1"),
					testAccCheckAIConfigVariationToolsAttached(resourceName, toolKey),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigVariationWithToolKeysUpdate, toolKey, secondToolKey, configKey, variationKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigVariationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "tool_keys.#", "2"),
					testAccCheckAIConfigVariationToolsAttached(resourceName, toolKey, secondToolKey),
				),
			},
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigVariationWithToolKeysRemoved, toolKey, secondToolKey, configKey, variationKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigVariationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "tool_keys.#", "0"),
					testAccCheckAIConfigVariationToolsAttached(resourceName),
				),
			},
		},
	})
}

func TestAccAIConfigVariation_WithInlineModel(t *testing.T) {
	aiTestCooldown()
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	configKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	variationKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_config_variation.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAIConfigVariationDestroy,
		Steps: []resource.TestStep{
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigVariationWithInlineModel, configKey, variationKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigVariationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Variation with inline model"),
					resource.TestCheckResourceAttrSet(resourceName, MODEL),
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

// testAccCheckAIConfigVariationToolsAttached asserts against the API — not
// Terraform state — that the variation has exactly the expected tools
// attached. State-only checks (tool_keys.#) are masked by the read's
// preserve-prior fallback, which echoes the planned tool_keys back into
// state when the API returns no tools; this check cannot be masked.
func testAccCheckAIConfigVariationToolsAttached(resourceName string, expected ...string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		configKey := rs.Primary.Attributes[AI_CONFIG_KEY]
		variationKey := rs.Primary.Attributes[KEY]

		client := mustTestAccClient()
		variationsResp, _, err := client.ld.AgentControlApi.GetAIConfigVariation(client.ctx, projectKey, configKey, variationKey).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting AI config variation: %s", err)
		}
		if variationsResp == nil || len(variationsResp.Items) == 0 {
			return fmt.Errorf("no variation versions returned for %s/%s/%s", projectKey, configKey, variationKey)
		}
		variation := variationsResp.Items[0]
		for _, v := range variationsResp.Items[1:] {
			if v.Version > variation.Version {
				variation = v
			}
		}
		got := make([]string, 0, len(variation.Tools))
		for _, t := range variation.Tools {
			got = append(got, t.Key)
		}
		sort.Strings(got)
		want := append([]string{}, expected...)
		sort.Strings(want)
		if !reflect.DeepEqual(got, want) {
			return fmt.Errorf("API reports attached tools %v, want %v — tool_keys accepted on write but not attached or not returned on read", got, want)
		}
		return nil
	}
}

func testAccCheckAIConfigVariationExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("AI config variation ID is not set")
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		configKey := rs.Primary.Attributes[AI_CONFIG_KEY]
		variationKey := rs.Primary.Attributes[KEY]

		client := mustTestAccClient()
		_, _, err := client.ld.AgentControlApi.GetAIConfigVariation(client.ctx, projectKey, configKey, variationKey).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting AI config variation: %s", err)
		}
		return nil
	}
}

var testAccCheckAIConfigVariationDestroy = func(s *terraform.State) error {
	client := mustTestAccClient()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_ai_config_variation" {
			continue
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		configKey := rs.Primary.Attributes[AI_CONFIG_KEY]
		variationKey := rs.Primary.Attributes[KEY]

		_, res, err := client.ld.AgentControlApi.GetAIConfigVariation(client.ctx, projectKey, configKey, variationKey).Execute()
		if isStatusNotFound(res) {
			continue
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("AI config variation %s/%s/%s still exists", projectKey, configKey, variationKey)
	}
	return nil
}
