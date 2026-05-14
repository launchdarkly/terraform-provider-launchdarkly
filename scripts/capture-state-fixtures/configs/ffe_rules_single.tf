# Synthetic capture config for launchdarkly_feature_flag_environment
# with one rule + clause.

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
  key  = "fixture-ffe-rules-pj"
  name = "Phase 4 FFE rules project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_feature_flag" "flag" {
  project_key    = launchdarkly_project.test.key
  key            = "fixture-rules-flag"
  name           = "Phase 4 FFE rules flag fixture"
  variation_type = "string"
  variations {
    value = "alpha"
  }
  variations {
    value = "beta"
  }
}

resource "launchdarkly_feature_flag_environment" "rules" {
  flag_id       = launchdarkly_feature_flag.flag.id
  env_key       = "fixture-env-test"
  on            = true
  off_variation = 1
  rules {
    description = "in-US"
    variation   = 0
    clauses {
      attribute = "country"
      op        = "in"
      values    = ["US"]
    }
  }
  fallthrough {
    variation = 1
  }
}
