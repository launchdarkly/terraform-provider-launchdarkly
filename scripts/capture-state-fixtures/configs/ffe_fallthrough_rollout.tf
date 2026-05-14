# Synthetic capture config for launchdarkly_feature_flag_environment
# with fallthrough.rollout_weights + bucket_by + context_kind.

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
  key  = "fixture-ffe-fallrollout-pj"
  name = "Phase 4 FFE fallthrough-rollout project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_feature_flag" "flag" {
  project_key    = launchdarkly_project.test.key
  key            = "fixture-fallrollout-flag"
  name           = "Phase 4 FFE fallthrough rollout flag fixture"
  variation_type = "string"
  variations {
    value = "alpha"
  }
  variations {
    value = "beta"
  }
  variations {
    value = "gamma"
  }
}

resource "launchdarkly_feature_flag_environment" "fallrollout" {
  flag_id       = launchdarkly_feature_flag.flag.id
  env_key       = "fixture-env-test"
  on            = true
  off_variation = 2
  fallthrough {
    rollout_weights = [33333, 33334, 33333]
    bucket_by       = "email"
    context_kind    = "user"
  }
}
