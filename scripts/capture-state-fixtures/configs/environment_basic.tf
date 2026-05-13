# Synthetic capture config for launchdarkly_environment (Phase 3.4).
# Pinned to v2.29.0 SDKv2 release; do NOT update the version without
# also re-capturing the fixture. See
# scripts/capture-state-fixtures/README.md and the fixture-safety
# policy in MIGRATION_PLAN_NON_BREAKING.md §Phase 0.5.

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
  name = "Phase 3 environment fixture"

  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_environment" "basic" {
  project_key = launchdarkly_project.test.key
  key         = "fixture-env-prod"
  name        = "Fixture prod env"
  color       = "445566"
  default_ttl = 5
  tags        = ["fixture"]

  approval_settings {
    required          = true
    min_num_approvals = 1
  }
}
