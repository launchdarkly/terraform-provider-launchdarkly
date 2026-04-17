package launchdarkly

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

const (
	testAccAIConfigCreate = `
resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "%s"
	description = "%s"
	tags        = ["test"]
}
`

	testAccAIConfigUpdate = `
resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "%s"
	description = "%s"
	tags        = ["test", "updated"]
}
`

	testAccAIConfigWithMode = `
resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "Agent Mode Config"
	description = "Agent mode AI config"
	mode        = "agent"
}
`

	testAccAIConfigWithMaintainer = `
resource "launchdarkly_ai_config" "test" {
	project_key   = launchdarkly_project.test.key
	key           = "%s"
	name          = "Maintained AI Config"
	description   = "AI config with member maintainer"
	maintainer_id = "%s"
}
`

	testAccAIConfigWithTeamMaintainer = `
resource "launchdarkly_team" "test" {
	key              = "%s"
	name             = "AI Config Test Team"
	custom_role_keys = []
	depends_on       = [launchdarkly_project.test]
}

resource "launchdarkly_ai_config" "test" {
	project_key         = launchdarkly_project.test.key
	key                 = "%s"
	name                = "Team Maintained AI Config"
	description         = "AI config with team maintainer"
	maintainer_team_key = launchdarkly_team.test.key
}
`

	testAccAIConfigWithEvaluationMetric = `
resource "launchdarkly_ai_config" "test" {
	project_key           = launchdarkly_project.test.key
	key                   = "%s"
	name                  = "Evaluated AI Config"
	description           = "AI config with evaluation metric"
	mode                  = "judge"
	evaluation_metric_key = "$ld:ai:judge:%s"
	is_inverted           = %t
}
`

	testAccAIConfigRemoveOptionals = `
resource "launchdarkly_ai_config" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "%s"
}
`
)

func TestAccAIConfig_CreateAndUpdate(t *testing.T) {
	aiTestCooldown()
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	configKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	configName := "Test AI Config"
	configDescription := "Test AI config description"
	updatedConfigName := "Updated Test AI Config"
	updatedConfigDescription := "Updated AI config description"
	resourceName := "launchdarkly_ai_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAIConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigCreate, configKey, configName, configDescription)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, KEY, configKey),
					resource.TestCheckResourceAttr(resourceName, NAME, configName),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, configDescription),
					resource.TestCheckResourceAttr(resourceName, MODE, "completion"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, VERSION),
					resource.TestCheckResourceAttrSet(resourceName, CREATION_DATE),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigUpdate, configKey, updatedConfigName, updatedConfigDescription)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, updatedConfigName),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, updatedConfigDescription),
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

func TestAccAIConfig_WithMode(t *testing.T) {
	aiTestCooldown()
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	configKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAIConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigWithMode, configKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, MODE, "agent"),
					resource.TestCheckResourceAttr(resourceName, NAME, "Agent Mode Config"),
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

func TestAccAIConfig_WithMaintainer(t *testing.T) {
	aiTestCooldown()
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	configKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_config.test"

	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)

	members, _, err := client.ld.AccountMembersApi.GetMembers(client.ctx).Execute()
	require.NoError(t, err)
	require.True(t, len(members.Items) > 0, "This test requires at least one member in the account")
	maintainerId := members.Items[0].Id

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAIConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigWithMaintainer, configKey, maintainerId)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, MAINTAINER_ID, maintainerId),
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

func TestAccAIConfig_WithTeamMaintainer(t *testing.T) {
	aiTestCooldown()
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	configKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	teamKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAIConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigWithTeamMaintainer, teamKey, configKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, MAINTAINER_TEAM_KEY, teamKey),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{MAINTAINER_TEAM_KEY},
			},
		},
	})
}

func TestAccAIConfig_WithEvaluationMetric(t *testing.T) {
	aiTestCooldown()
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	configKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	metricSuffix := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	evalMetricKey := "$ld:ai:judge:" + metricSuffix
	resourceName := "launchdarkly_ai_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAIConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigWithEvaluationMetric, configKey, metricSuffix, false)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, EVALUATION_METRIC_KEY, evalMetricKey),
					resource.TestCheckResourceAttr(resourceName, IS_INVERTED, "false"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigWithEvaluationMetric, configKey, metricSuffix, true)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, IS_INVERTED, "true"),
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

func TestAccAIConfig_RemoveOptionalFields(t *testing.T) {
	aiTestCooldown()
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	configKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_ai_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAIConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigCreate, configKey, "Full Config", "A description")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "A description"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
				),
			},
			{
				Config: withAITestProject(projectKey, fmt.Sprintf(testAccAIConfigRemoveOptionals, configKey, "Full Config")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAIConfigExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, ""),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "0"),
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

func TestShouldRetryAIConfigDelete(t *testing.T) {
	tests := []struct {
		name string
		res  *http.Response
		err  error
		want bool
	}{
		{
			name: "retries known transient ai config delete error",
			res:  &http.Response{StatusCode: http.StatusBadRequest},
			err:  errors.New(`400 Bad Request: {"code":"invalid_request","message":"could not delete AI Config: abc123"}`),
			want: true,
		},
		{
			name: "handles case-insensitive message",
			res:  &http.Response{StatusCode: http.StatusBadRequest},
			err:  errors.New(`400 Bad Request: {"code":"invalid_request","message":"Could Not Delete Ai Config: abc123"}`),
			want: true,
		},
		{
			name: "does not retry unrelated bad request",
			res:  &http.Response{StatusCode: http.StatusBadRequest},
			err:  errors.New(`400 Bad Request: {"code":"invalid_request","message":"validation failed"}`),
			want: false,
		},
		{
			name: "does not retry non-400 response",
			res:  &http.Response{StatusCode: http.StatusConflict},
			err:  errors.New(`409 Conflict: {"code":"conflict","message":"resource conflict"}`),
			want: false,
		},
		{
			name: "does not retry without response",
			err:  errors.New("request failed"),
			want: false,
		},
		{
			name: "does not retry nil error",
			res:  &http.Response{StatusCode: http.StatusBadRequest},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldRetryAIConfigDelete(tc.res, tc.err)
			if got != tc.want {
				t.Fatalf("shouldRetryAIConfigDelete() = %v, want %v", got, tc.want)
			}
		})
	}
}

func testAccCheckAIConfigExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("AI config ID is not set")
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		configKey := rs.Primary.Attributes[KEY]

		client := testAccProvider.Meta().(*Client)
		_, _, err := client.ld.AIConfigsApi.GetAIConfig(client.ctx, projectKey, configKey).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting AI config: %s", err)
		}
		return nil
	}
}

var testAccCheckAIConfigDestroy = func(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_ai_config" {
			continue
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		configKey := rs.Primary.Attributes[KEY]

		_, res, err := client.ld.AIConfigsApi.GetAIConfig(client.ctx, projectKey, configKey).Execute()
		if isStatusNotFound(res) {
			continue
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("AI config %s/%s still exists", projectKey, configKey)
	}
	return nil
}
