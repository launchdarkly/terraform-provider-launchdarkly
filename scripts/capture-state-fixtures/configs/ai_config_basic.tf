# Synthetic capture config for launchdarkly_ai_config (Phase 3.5).
# mode=completion is the default; explicitly set so the captured state
# is deterministic across provider releases that might change the default.

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
  name = "Phase 3 ai_config fixture"

  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_ai_config" "basic" {
  project_key = launchdarkly_project.test.key
  key         = "fixture-ai-config"
  name        = "Fixture AI Config"
  description = "Synthetic AI config for state-compat capture."
  mode        = "completion"
  tags        = ["fixture"]
}
