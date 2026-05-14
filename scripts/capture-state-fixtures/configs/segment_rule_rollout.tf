# Synthetic capture config for launchdarkly_segment with a rule
# carrying weight + bucket_by + rollout_context_kind (the rollout path
# through segment_rule_helper.go).

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
  key  = "fixture-segment-rollout-pj"
  name = "Phase 4 segment rollout project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_segment" "rollout" {
  project_key = launchdarkly_project.test.key
  env_key     = "fixture-env-test"
  key         = "fixture-segment-rollout"
  name        = "Phase 4 segment rollout fixture"

  rules {
    weight               = 50000
    bucket_by            = "email"
    rollout_context_kind = "user"
    clauses {
      attribute = "country"
      op        = "in"
      values    = ["US"]
    }
  }
}
