# Synthetic capture config for launchdarkly_segment with included +
# excluded user-context lists.

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
  key  = "fixture-segment-incexc-pj"
  name = "Phase 4 segment incexc project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_segment" "incexc" {
  project_key = launchdarkly_project.test.key
  env_key     = "fixture-env-test"
  key         = "fixture-segment-incexc"
  name        = "Phase 4 segment incexc fixture"
  description = "Included + excluded user keys"
  tags        = ["fixture"]

  included = ["alice@example.invalid", "bob@example.invalid"]
  excluded = ["mallory@example.invalid"]
}
