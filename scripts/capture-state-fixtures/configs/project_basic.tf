# Synthetic capture config for launchdarkly_project (Phase 4.1).
# Minimal: 1 env, no IIS / CSA / approval_settings. Tests the
# "neither IIS nor CSA declared" branch of customizeProjectDiff →
# ModifyPlan.

terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "2.29.0"
    }
  }
}

provider "launchdarkly" {}

resource "launchdarkly_project" "basic" {
  key  = "fixture-project-basic"
  name = "Phase 4 project basic fixture"

  environments {
    key   = "fixture-env-basic"
    name  = "Fixture env"
    color = "112233"
  }
}
