# Synthetic capture config for launchdarkly_feature_flag_environment
# with non-user context targets.

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
  key  = "fixture-ffe-ctxtgt-pj"
  name = "Phase 4 FFE ctx-targets project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_feature_flag" "flag" {
  project_key    = launchdarkly_project.test.key
  key            = "fixture-ctxtgt-flag"
  name           = "Phase 4 FFE ctx-targets flag fixture"
  variation_type = "boolean"
  variations {
    value = "true"
  }
  variations {
    value = "false"
  }
}

resource "launchdarkly_feature_flag_environment" "ctx_targets" {
  flag_id       = launchdarkly_feature_flag.flag.id
  env_key       = "fixture-env-test"
  on            = true
  off_variation = 1
  context_targets {
    context_kind = "organization"
    values       = ["acme", "globex"]
    variation    = 0
  }
  fallthrough {
    variation = 1
  }
}
