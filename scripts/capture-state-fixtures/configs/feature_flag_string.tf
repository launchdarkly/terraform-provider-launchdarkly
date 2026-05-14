# Synthetic capture config for launchdarkly_feature_flag string
# variation type.

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
  key  = "fixture-ff-string-pj"
  name = "Phase 4 ff string project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_feature_flag" "string_flag" {
  project_key    = launchdarkly_project.test.key
  key            = "fixture-string-flag"
  name           = "Phase 4 string flag fixture"
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
