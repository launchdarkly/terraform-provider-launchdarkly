resource "launchdarkly_model_config" "example" {
  project_key          = launchdarkly_project.example.key
  key                  = "gpt-4-turbo"
  name                 = "GPT-4 Turbo"
  model_id             = "gpt-4-turbo"
  provider             = "openai"
  cost_per_input_token = 0.00001
  tags                 = ["production"]
}
