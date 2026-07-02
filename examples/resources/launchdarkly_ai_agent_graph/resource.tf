resource "launchdarkly_ai_config" "triage_agent" {
  project_key = launchdarkly_project.example.key
  key         = "triage-agent"
  name        = "Triage agent"
}

resource "launchdarkly_ai_config" "support_agent" {
  project_key = launchdarkly_project.example.key
  key         = "support-agent"
  name        = "Support agent"
  depends_on  = [launchdarkly_ai_config.triage_agent]
}

resource "launchdarkly_ai_agent_graph" "support_workflow" {
  project_key     = launchdarkly_project.example.key
  key             = "support-workflow"
  name            = "Support workflow"
  description     = "Routes incoming requests from the triage agent to the support agent"
  root_config_key = launchdarkly_ai_config.triage_agent.key
  edges = {
    "triage-to-support" = {
      source_config = launchdarkly_ai_config.triage_agent.key
      target_config = launchdarkly_ai_config.support_agent.key
      handoff       = jsonencode({ reason = "needs_human_support" })
    }
  }
}
