data "launchdarkly_ai_config" "example" {
  project_key = "example-project"
  key         = "example-ai-config"
}

output "ai_config_details" {
  value = {
    name        = data.launchdarkly_ai_config.example.name
    description = data.launchdarkly_ai_config.example.description
    mode        = data.launchdarkly_ai_config.example.mode
  }
}
