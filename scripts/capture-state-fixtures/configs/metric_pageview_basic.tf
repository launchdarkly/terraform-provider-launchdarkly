# Synthetic capture config for launchdarkly_metric (kind=pageview)
# (Phase 3.3). Exercises the urls block so the framework's
# customizeMetricDiff port + Map<>List conversion is covered.

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
  name = "Phase 3 metric (pageview) fixture"

  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_metric" "pageview_basic" {
  project_key = launchdarkly_project.test.key
  key         = "fixture-metric-pageview"
  name        = "Fixture pageview metric"
  description = "Synthetic pageview metric for state-compat capture."
  kind        = "pageview"
  tags        = ["fixture"]

  urls {
    kind = "exact"
    url  = "https://example.invalid/fixture-page"
  }

  randomization_units = ["user"]
}
