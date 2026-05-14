# Synthetic capture config for launchdarkly_feature_flag with the
# deprecated include_in_snippet attribute declared (mutual ConflictsWith
# with client_side_availability).

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
  key  = "fixture-ff-iis-pj"
  name = "Phase 4 ff iis project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_feature_flag" "iis_flag" {
  project_key        = launchdarkly_project.test.key
  key                = "fixture-iis-flag"
  name               = "Phase 4 iis flag fixture"
  variation_type     = "boolean"
  include_in_snippet = true

  variations {
    value = "true"
  }
  variations {
    value = "false"
  }
}
