package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGeneratedAccountRelayAutoConfig_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_account_relay_auto_config.")
}
func TestAccGeneratedApplication_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_application.")
}
func TestAccGeneratedApprovalRequest_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_approval_request.")
}
func TestAccGeneratedApprovalRequestProjectSetting_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_approval_request_project_setting.")
}
func TestAccGeneratedAuditlog_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_auditlog.")
}
func TestAccGeneratedCodeRefRepository_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_code_ref_repository.")
}
func TestAccGeneratedCodeRefRepositoryBranche_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_code_ref_repository_branche.")
}
func TestAccGeneratedDestination_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_destination.")
}
func TestAccGeneratedEngineeringInsightDeployment_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_engineering_insight_deployment.")
}
func TestAccGeneratedEngineeringInsightInsightGroup_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_engineering_insight_insight_group.")
}
func TestAccGeneratedFlag_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_flag.")
}
func TestAccGeneratedFlagExpiringTarget_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_flag_expiring_target.")
}
func TestAccGeneratedFlagExpiringUserTarget_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_flag_expiring_user_target.")
}
func TestAccGeneratedFlagRelease_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_flag_release.")
}
func TestAccGeneratedFlagTrigger_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_flag_trigger.")
}
func TestAccGeneratedIntegration_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_integration.")
}
func TestAccGeneratedIntegrationCapabilityBigSegmentStore_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_integration_capability_big_segment_store.")
}
func TestAccGeneratedIntegrationCapabilityFeatureStore_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_integration_capability_feature_store.")
}
func TestAccGeneratedIntegrationCapabilityFlagImport_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_integration_capability_flag_import.")
}
func TestAccGeneratedIntegrationConfiguration_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_integration_configuration.")
}
func TestAccGeneratedIntegrationConfigurationKey_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_integration_configuration_key.")
}
func TestAccGeneratedMember_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_member.")
}
func TestAccGeneratedMetric_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_metric.")
}
func TestAccGeneratedOauthClient_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_oauth_client.")
}
func testAccGeneratedProjectConfigBasic(projectKey string) string {
	return fmt.Sprintf(`
resource "launchdarkly_generated_project" "generated" {
  key                = %q
  name               = "Generated Project"
  include_in_snippet = false
  tags               = ["generated", "baseline"]

  environments {
    key   = "generated-env"
    name  = "Generated Env"
    color = "010101"
  }
}
`, projectKey)
}

func testAccGeneratedProjectConfigUpdate(projectKey string) string {
	return fmt.Sprintf(`
resource "launchdarkly_generated_project" "generated" {
  key                = %q
  name               = "Generated Project Updated"
  include_in_snippet = true
  tags               = ["generated", "updated"]

  environments {
    key                  = "generated-env"
    name                 = "Generated Env Updated"
    color                = "020202"
    default_ttl          = 30
    secure_mode          = true
    default_track_events = true
    require_comments     = true
    confirm_changes      = true
  }
}
`, projectKey)
}

func testAccGeneratedProjectConfigRevert(projectKey string) string {
	return fmt.Sprintf(`
resource "launchdarkly_generated_project" "generated" {
  key  = %q
  name = "Generated Project Updated"

  environments {
    key   = "generated-env"
    name  = "Generated Env Updated"
    color = "020202"
  }
}
`, projectKey)
}

func TestAccGeneratedProject_basic(t *testing.T) {
	t.Parallel()

	resourceName := "launchdarkly_generated_project.generated"
	projectKey := "gen-project-" + acctest.RandStringFromCharSet(12, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccFrameworkMuxProviders(context.Background(), t),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccGeneratedProjectConfigBasic(projectKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "name", "Generated Project"),
					resource.TestCheckResourceAttr(resourceName, "include_in_snippet", "false"),
					resource.TestCheckResourceAttr(resourceName, "default_client_side_availability.0.using_environment_id", "false"),
					resource.TestCheckResourceAttr(resourceName, "default_client_side_availability.0.using_mobile_key", "true"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "environments.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.name", "Generated Env"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccGeneratedProjectConfigUpdate(projectKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Generated Project Updated"),
					resource.TestCheckResourceAttr(resourceName, "include_in_snippet", "true"),
					resource.TestCheckResourceAttr(resourceName, "default_client_side_availability.0.using_environment_id", "true"),
					resource.TestCheckResourceAttr(resourceName, "default_client_side_availability.0.using_mobile_key", "true"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.name", "Generated Env Updated"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.default_ttl", "30"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.secure_mode", "true"),
				),
			},
			{
				Config: testAccGeneratedProjectConfigRevert(projectKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "include_in_snippet", "false"),
					resource.TestCheckResourceAttr(resourceName, "default_client_side_availability.0.using_environment_id", "false"),
					resource.TestCheckResourceAttr(resourceName, "default_client_side_availability.0.using_mobile_key", "true"),
					resource.TestCheckNoResourceAttr(resourceName, "tags.#"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.default_ttl", "0"),
					resource.TestCheckResourceAttr(resourceName, "environments.0.secure_mode", "false"),
				),
			},
		},
	})
}
func TestAccGeneratedProjectAgentGraph_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_project_agent_graph.")
}
func TestAccGeneratedProjectAgentOptimization_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_project_agent_optimization.")
}
func TestAccGeneratedProjectAiConfig_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_project_ai_config.")
}
func TestAccGeneratedProjectAiConfigModelConfig_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_project_ai_config_model_config.")
}
func TestAccGeneratedProjectAiConfigPromptSnippet_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_project_ai_config_prompt_snippet.")
}
func TestAccGeneratedProjectAiConfigTargeting_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_project_ai_config_targeting.")
}
func TestAccGeneratedProjectAiConfigVariation_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_project_ai_config_variation.")
}
func TestAccGeneratedProjectAiTool_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_project_ai_tool.")
}
func TestAccGeneratedProjectEnvironment_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_project_environment.")
}
func TestAccGeneratedProjectEnvironmentContextInstance_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_project_environment_context_instance.")
}
func TestAccGeneratedProjectEnvironmentExperiment_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_project_environment_experiment.")
}
func TestAccGeneratedProjectEnvironmentHoldout_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_project_environment_holdout.")
}
func TestAccGeneratedProjectExperimentationSetting_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_project_experimentation_setting.")
}
func TestAccGeneratedProjectFlagDefault_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_project_flag_default.")
}
func TestAccGeneratedProjectFlagEnvironmentApprovalRequest_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_project_flag_environment_approval_request.")
}
func TestAccGeneratedProjectFlagEnvironmentScheduledChange_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_project_flag_environment_scheduled_change.")
}
func TestAccGeneratedProjectFlagEnvironmentWorkflow_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_project_flag_environment_workflow.")
}
func TestAccGeneratedProjectMetricGroup_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_project_metric_group.")
}
func TestAccGeneratedProjectReleasePipeline_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_project_release_pipeline.")
}
func TestAccGeneratedProjectReleasePolicy_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_project_release_policy.")
}
func TestAccGeneratedProjectView_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_project_view.")
}
func TestAccGeneratedRole_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_role.")
}
func TestAccGeneratedSegment_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_segment.")
}
func TestAccGeneratedSegmentContext_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_segment_context.")
}
func TestAccGeneratedSegmentExpiringTarget_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_segment_expiring_target.")
}
func TestAccGeneratedSegmentExpiringUserTarget_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_segment_expiring_user_target.")
}
func TestAccGeneratedSegmentExport_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_segment_export.")
}
func TestAccGeneratedSegmentImport_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_segment_import.")
}
func TestAccGeneratedSegmentUser_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_segment_user.")
}
func testAccGeneratedTeamConfigBasic(teamKey string) string {
	return fmt.Sprintf(`
resource "launchdarkly_generated_team" "generated" {
  key         = %q
  name        = "Generated Team"
  description = "generated-team-basic"
}
`, teamKey)
}

func testAccGeneratedTeamConfigUpdate(teamKey string) string {
	return fmt.Sprintf(`
resource "launchdarkly_generated_team" "generated" {
  key         = %q
  name        = "Generated Team"
  description = "generated-team-update"
}
`, teamKey)
}

func TestAccGeneratedTeam_basic(t *testing.T) {
	t.Parallel()

	resourceName := "launchdarkly_generated_team.generated"
	teamKey := "gen-team-" + acctest.RandStringFromCharSet(12, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccFrameworkMuxProviders(context.Background(), t),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccGeneratedTeamConfigBasic(teamKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", teamKey),
					resource.TestCheckResourceAttr(resourceName, "name", "Generated Team"),
					resource.TestCheckResourceAttr(resourceName, "description", "generated-team-basic"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccGeneratedTeamConfigUpdate(teamKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "generated-team-update"),
				),
			},
		},
	})
}
func TestAccGeneratedTeamRoleMapping_basic(t *testing.T) {
	t.Parallel()

	resourceName := "launchdarkly_generated_team_role_mapping.generated"
	role0 := "gen-role-0-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	role1 := "gen-role-1-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	teamKey := "gen-team-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccFrameworkMuxProviders(context.Background(), t),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
%s
resource "launchdarkly_generated_team_role_mapping" "generated" {
  team_key = launchdarkly_team.test_team.key
  custom_role_keys = [
    launchdarkly_custom_role.role_0.key,
    launchdarkly_custom_role.role_1.key,
  ]
}
`, testAccTeamRoleMappingSetup(role0, role1, teamKey)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "team_key", teamKey),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.0", role0),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.1", role1),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(`
%s
resource "launchdarkly_generated_team_role_mapping" "generated" {
  team_key = launchdarkly_team.test_team.key
  custom_role_keys = [launchdarkly_custom_role.role_1.key]
}
`, testAccTeamRoleMappingSetup(role0, role1, teamKey)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "team_key", teamKey),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.0", role1),
				),
			},
		},
	})
}
func TestAccGeneratedToken_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_token.")
}
func TestAccGeneratedUser_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_user.")
}
func TestAccGeneratedUserExpiringUserTarget_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_user_expiring_user_target.")
}
func TestAccGeneratedUserFlag_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_user_flag.")
}
func TestAccGeneratedWebhook_generic(t *testing.T) {
	t.Parallel()
	t.Skip("Generic generated acceptance test requires test.fixture overlay for launchdarkly_generated_webhook.")
}
