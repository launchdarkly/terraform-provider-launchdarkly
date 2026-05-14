# Synthetic capture config for launchdarkly_feature_flag boolean
# variation type with explicit variations.

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
  key  = "fixture-ff-bool-pj"
  name = "Phase 4 ff bool project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_feature_flag" "bool_flag" {
  project_key    = launchdarkly_project.test.key
  key            = "fixture-bool-flag"
  name           = "Phase 4 boolean flag fixture"
  variation_type = "boolean"

  variations {
    value = "true"
    name  = "On"
  }
  variations {
    value = "false"
    name  = "Off"
  }
}
