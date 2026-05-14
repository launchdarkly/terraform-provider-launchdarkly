# Synthetic capture config for launchdarkly_team (Phase 3.7). Uses
# an empty members list and no custom_role_keys so the capture
# produces deterministic state without a separate Member fixture.
# Note: teams are account-scoped, so the team key must be globally
# unique within the test LD account. The fixture-team-* prefix is
# deliberately chosen so it's obvious in audit logs.

terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "2.29.0"
    }
  }
}

provider "launchdarkly" {}

resource "launchdarkly_team" "basic" {
  key         = "fixture-team-basic"
  name        = "Fixture team"
  description = "Synthetic team for state-compat capture."

  member_ids       = []
  maintainers      = []
  custom_role_keys = []
}
