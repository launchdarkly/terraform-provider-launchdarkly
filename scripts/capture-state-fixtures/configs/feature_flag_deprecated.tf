# Synthetic capture config for launchdarkly_feature_flag with the
# v2.29-era `deprecated = true` attribute. Tests the deprecated-bool
# nil-safety path that landed in commit 48441122.

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
  key  = "fixture-ff-dep-pj"
  name = "Phase 4 ff deprecated project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_feature_flag" "deprecated_flag" {
  project_key    = launchdarkly_project.test.key
  key            = "fixture-deprecated-flag"
  name           = "Phase 4 deprecated flag fixture"
  variation_type = "boolean"
  deprecated     = true

  variations {
    value = "true"
  }
  variations {
    value = "false"
  }
}
