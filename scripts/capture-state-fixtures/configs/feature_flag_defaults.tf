# Synthetic capture config for launchdarkly_feature_flag with explicit
# defaults block.

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
  key  = "fixture-ff-defaults-pj"
  name = "Phase 4 ff defaults project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_feature_flag" "defaults_flag" {
  project_key    = launchdarkly_project.test.key
  key            = "fixture-defaults-flag"
  name           = "Phase 4 defaults flag fixture"
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

  defaults {
    on_variation  = 1
    off_variation = 2
  }
}
