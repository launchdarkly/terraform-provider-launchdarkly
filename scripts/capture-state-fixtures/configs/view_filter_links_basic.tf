# Synthetic capture config for launchdarkly_view_filter_links
# (Phase 3.8). Filter-based linking using a flag tag, so the fixture
# is independent of which specific flag keys exist in the LD account.

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
  name = "Phase 3 view_filter_links fixture"

  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_view" "linked" {
  project_key = launchdarkly_project.test.key
  key         = "fixture-view-filter-1"
  name        = "Fixture filter view"
}

resource "launchdarkly_view_filter_links" "basic" {
  project_key        = launchdarkly_project.test.key
  view_key           = launchdarkly_view.linked.key
  flag_filter        = "tags:fixture"
  reconcile_on_apply = false
}
