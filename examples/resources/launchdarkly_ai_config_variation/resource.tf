resource "launchdarkly_ai_config_variation" "example" {
  project_key      = launchdarkly_project.example.key
  config_key       = launchdarkly_ai_config.example.key
  key              = "helpful-v1"
  name             = "Helpful V1"
  model_config_key = launchdarkly_model_config.example.key

  messages {
    role    = "system"
    content = "You are a helpful customer support assistant."
  }

  messages {
    role    = "user"
    content = "{{ ldctx.query }}"
  }
}
