# Synthetic capture config for launchdarkly_feature_flag with
# custom_properties. Tests the customPropertyHash sort parity.

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
  key  = "fixture-ff-cp-pj"
  name = "Phase 4 ff custom-props project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_feature_flag" "cp_flag" {
  project_key    = launchdarkly_project.test.key
  key            = "fixture-cp-flag"
  name           = "Phase 4 custom-props flag fixture"
  variation_type = "boolean"

  variations {
    value = "true"
  }
  variations {
    value = "false"
  }

  custom_properties {
    key   = "fixture-ticket"
    name  = "Ticket"
    value = ["JIRA-100", "JIRA-200"]
  }
  custom_properties {
    key   = "fixture-team"
    name  = "Team"
    value = ["platform"]
  }
}
