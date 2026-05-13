# Synthetic capture config for launchdarkly_view_links (Phase 3.8).
# Uses a view + a single feature flag inside the same project so the
# fixture exercises explicit-link tracking without depending on
# segments.
#
# NOTE: launchdarkly_view is a beta resource served by the framework
# already (Phase 1.3.4 for the data source; the resource side ships
# pre-Phase 3). It pins the same v2.29.0 SDKv2 line as the rest of
# Phase 3 for legacy-state capture.

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
  key  = "fixture-project-1"
  name = "Phase 3 view_links fixture"

  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_view" "linked" {
  project_key = launchdarkly_project.test.key
  key         = "fixture-view-1"
  name        = "Fixture view"
}

resource "launchdarkly_feature_flag" "linked" {
  project_key    = launchdarkly_project.test.key
  key            = "fixture-flag-1"
  name           = "Fixture flag"
  variation_type = "boolean"
}

resource "launchdarkly_view_links" "basic" {
  project_key = launchdarkly_project.test.key
  view_key    = launchdarkly_view.linked.key

  flags = [launchdarkly_feature_flag.linked.key]
}
