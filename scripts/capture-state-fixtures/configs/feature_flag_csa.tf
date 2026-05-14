# Synthetic capture config for launchdarkly_feature_flag with
# client_side_availability declared.

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
  key  = "fixture-ff-csa-pj"
  name = "Phase 4 ff csa project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_feature_flag" "csa_flag" {
  project_key    = launchdarkly_project.test.key
  key            = "fixture-csa-flag"
  name           = "Phase 4 csa flag fixture"
  variation_type = "boolean"

  client_side_availability {
    using_environment_id = true
    using_mobile_key     = false
  }

  variations {
    value = "true"
  }
  variations {
    value = "false"
  }
}
