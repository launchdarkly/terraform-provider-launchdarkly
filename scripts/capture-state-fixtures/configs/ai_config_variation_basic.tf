# Synthetic capture config for launchdarkly_ai_config_variation
# (Phase 3.6). Variations are versioned (PATCH creates a new version),
# so the fixture should be deterministic on a fresh project — the
# capture flow tears down via terraform destroy after applying so the
# project key is reused only within a single capture pass.

terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "2.29.0"
    }
  }
}

provider "launchdarkly" {}

resource "launchdarkly_project" "test" {
  key  = "fixture-project-1"
  name = "Phase 3 ai_config_variation fixture"

  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_ai_config" "parent" {
  project_key = launchdarkly_project.test.key
  key         = "fixture-ai-config-parent"
  name        = "Fixture AI Config (parent)"
  mode        = "completion"
}

resource "launchdarkly_ai_config_variation" "basic" {
  project_key = launchdarkly_project.test.key
  config_key  = launchdarkly_ai_config.parent.key
  key         = "fixture-variation-1"
  name        = "Fixture variation"
  description = "Synthetic variation for state-compat capture."

  model = jsonencode({
    modelName = "synthetic-model"
    parameters = {
      temperature = 0.5
    }
  })

  messages {
    role    = "system"
    content = "You are a fixture."
  }
  messages {
    role    = "user"
    content = "Hello fixture."
  }
}
