terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "2.29.0"
    }
  }
}

resource "launchdarkly_project" "fixture" {
  key  = "fixture-project-1"
  name = "fixture-project-1"
}

resource "launchdarkly_model_config" "fixture" {
  project_key   = launchdarkly_project.fixture.key
  key           = "fixture-model-config-basic"
  name          = "fixture-model-config-basic"
  provider_name = "openai"
  id            = "gpt-4o-mini"
  params = jsonencode({
    temperature = 0.7
    maxTokens   = 4096
  })
}
