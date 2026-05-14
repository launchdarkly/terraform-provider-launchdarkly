# Synthetic capture config for launchdarkly_project with a nested
# environment that declares approval_settings. Tests the shared
# approval-settings block plumbing through the project resource.

terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "2.29.0"
    }
  }
}

provider "launchdarkly" {}

resource "launchdarkly_project" "env_approvals" {
  key  = "fixture-project-envapprovals"
  name = "Phase 4 project env-approvals fixture"

  environments {
    key   = "fixture-env-approvals"
    name  = "Fixture env with approvals"
    color = "112233"
    approval_settings {
      required                   = true
      min_num_approvals          = 2
      can_review_own_request     = false
      can_apply_declined_changes = false
    }
  }
}
