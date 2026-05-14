# Synthetic capture config for launchdarkly_feature_flag number
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
  key  = "fixture-ff-number-pj"
  name = "Phase 4 ff number project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_feature_flag" "number_flag" {
  project_key    = launchdarkly_project.test.key
  key            = "fixture-number-flag"
  name           = "Phase 4 number flag fixture"
  variation_type = "number"

  variations {
    value = "0"
  }
  variations {
    value = "42"
  }
  variations {
    value = "100"
  }
}
