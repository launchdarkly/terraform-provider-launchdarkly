# Synthetic capture config for launchdarkly_segment in big-segment
# mode (unbounded = true). Tests the ForceNew unbounded + unbounded
# context-kind path.

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
  key  = "fixture-segment-big-pj"
  name = "Phase 4 segment big project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_segment" "big" {
  project_key            = launchdarkly_project.test.key
  env_key                = "fixture-env-test"
  key                    = "fixture-segment-big"
  name                   = "Phase 4 segment big fixture"
  unbounded              = true
  unbounded_context_kind = "user"
}
