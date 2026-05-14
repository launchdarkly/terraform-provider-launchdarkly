# Synthetic capture config for launchdarkly_segment with a single
# rule + clause (no rollout). Tests the rule/clause schema port.

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
  key  = "fixture-segment-rules-pj"
  name = "Phase 4 segment rules project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_segment" "rules" {
  project_key = launchdarkly_project.test.key
  env_key     = "fixture-env-test"
  key         = "fixture-segment-rules"
  name        = "Phase 4 segment rules fixture"

  rules {
    clauses {
      attribute = "country"
      op        = "in"
      values    = ["US", "CA"]
    }
  }
}
