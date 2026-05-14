# Synthetic capture config for launchdarkly_feature_flag_environment
# with an explicit non-default off_variation index. Tests the
# off_variation Required Int handling.

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
  key  = "fixture-ffe-offvar-pj"
  name = "Phase 4 FFE off-variation project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_feature_flag" "flag" {
  project_key    = launchdarkly_project.test.key
  key            = "fixture-offvar-flag"
  name           = "Phase 4 FFE off-variation flag fixture"
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

resource "launchdarkly_feature_flag_environment" "offvar" {
  flag_id       = launchdarkly_feature_flag.flag.id
  env_key       = "fixture-env-test"
  on            = false
  off_variation = 0
  track_events  = true
  fallthrough {
    variation = 1
  }
}
