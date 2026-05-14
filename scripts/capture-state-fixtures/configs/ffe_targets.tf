# Synthetic capture config for launchdarkly_feature_flag_environment
# with explicit user-context targets.

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
  key  = "fixture-ffe-targets-pj"
  name = "Phase 4 FFE targets project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_feature_flag" "flag" {
  project_key    = launchdarkly_project.test.key
  key            = "fixture-targets-flag"
  name           = "Phase 4 FFE targets flag fixture"
  variation_type = "boolean"
  variations {
    value = "true"
  }
  variations {
    value = "false"
  }
}

resource "launchdarkly_feature_flag_environment" "targets" {
  flag_id       = launchdarkly_feature_flag.flag.id
  env_key       = "fixture-env-test"
  on            = true
  off_variation = 1
  targets {
    values    = ["fixture-user-1", "fixture-user-2"]
    variation = 0
  }
  targets {
    values    = ["fixture-user-9"]
    variation = 1
  }
  fallthrough {
    variation = 1
  }
}
