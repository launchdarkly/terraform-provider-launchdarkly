resource "launchdarkly_ai_config_variation" "example" {
  project_key      = launchdarkly_project.example.key
  config_key       = launchdarkly_ai_config.example.key
  key              = "helpful-v1"
  name             = "Helpful V1"
  model_config_key = launchdarkly_model_config.example.key

  messages = [
    {
      role    = "system"
      content = "You are a helpful customer support assistant."
    },
    {
      role    = "user"
      content = "{{ ldctx.query }}"
    },
  ]

  judges = {
    (launchdarkly_ai_config.response_quality_judge.key) = {
      sampling_rate = 0.1
    }
  }
}

resource "launchdarkly_ai_config" "response_quality_judge" {
  project_key = launchdarkly_project.example.key
  key         = "response-quality-judge"
  name        = "Response Quality Judge"
  mode        = "judge"
}
