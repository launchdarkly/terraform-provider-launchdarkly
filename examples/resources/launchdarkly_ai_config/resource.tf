resource "launchdarkly_ai_config" "example" {
  project_key = launchdarkly_project.example.key
  key         = "example-ai-config"
  name        = "Example AI Config"
  description = "An example AI config for managing AI-powered features."
  mode        = "completion"

  tags = [
    "terraform",
    "example"
  ]

  evaluation_metric_key = "example-metric"
  is_inverted           = false
  maintainer_id         = "507f1f77bcf86cd799439011"
}

# Alternative example with team maintainer
resource "launchdarkly_ai_config" "team_maintained" {
  project_key         = launchdarkly_project.example.key
  key                 = "team-ai-config"
  name                = "Team AI Config"
  mode                = "agent"
  maintainer_team_key = "platform-team"

  tags = ["team-managed"]
}
