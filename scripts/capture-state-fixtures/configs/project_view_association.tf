# Synthetic capture config for launchdarkly_project with view
# association requirements enabled. Exercises the raw HTTP patch path
# that lives outside the official API client model.

terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "2.29.0"
    }
  }
}

provider "launchdarkly" {}

resource "launchdarkly_project" "view_req" {
  key                                       = "fixture-project-viewreq"
  name                                      = "Phase 4 project view-req fixture"
  require_view_association_for_new_flags    = true
  require_view_association_for_new_segments = true

  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}
