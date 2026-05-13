# Synthetic capture config for launchdarkly_metric (kind=custom)
# (Phase 3.3). Exercises numeric=true so success_criteria + unit are
# both materialised in state.

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
  name = "Phase 3 metric (custom) fixture"

  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_metric" "custom_basic" {
  project_key      = launchdarkly_project.test.key
  key              = "fixture-metric-custom"
  name             = "Fixture custom metric"
  description      = "Synthetic metric for state-compat capture."
  kind             = "custom"
  is_numeric       = true
  unit             = "ms"
  event_key        = "fixture-event"
  success_criteria = "LowerThanBaseline"
  tags             = ["fixture"]

  randomization_units = ["user"]
}
