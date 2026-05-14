# Synthetic capture config for launchdarkly_project with explicit
# include_in_snippet (deprecated). Tests the IIS-only branch of
# customizeProjectDiff.

terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "2.29.0"
    }
  }
}

provider "launchdarkly" {}

resource "launchdarkly_project" "iis" {
  key                = "fixture-project-iis"
  name               = "Phase 4 project IIS fixture"
  include_in_snippet = true
  tags               = ["fixture", "iis"]

  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}
