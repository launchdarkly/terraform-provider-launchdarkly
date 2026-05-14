# Synthetic capture config for launchdarkly_project with explicit
# default_client_side_availability. Tests the CSA-only branch.

terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "2.29.0"
    }
  }
}

provider "launchdarkly" {}

resource "launchdarkly_project" "csa" {
  key  = "fixture-project-csa"
  name = "Phase 4 project CSA fixture"

  default_client_side_availability {
    using_environment_id = true
    using_mobile_key     = false
  }

  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}
