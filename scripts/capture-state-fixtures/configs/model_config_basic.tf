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
  environments {
    key   = "fixture-env-test"
    name  = "fixture-env-test"
    color = "AABBCC"
  }
}

resource "launchdarkly_model_config" "fixture" {
  project_key    = launchdarkly_project.fixture.key
  key            = "fixture-model-config-basic"
  name           = "fixture-model-config-basic"
  model_id       = "gpt-4o-mini"
  model_provider = "openai"
  params = jsonencode({
    temperature = 0.7
    maxTokens   = 4096
  })
}
