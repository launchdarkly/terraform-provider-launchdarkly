# Synthetic capture config for launchdarkly_ip_allowlist_config
# (Phase 3.9). This is a singleton; both flags default to false on
# create and Delete resets them to false rather than removing
# anything server-side, so capture-then-destroy is a no-op against
# the test account.

terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "2.29.0"
    }
  }
}

provider "launchdarkly" {}

resource "launchdarkly_ip_allowlist_config" "basic" {
  session_allowlist_enabled = false
  scoped_allowlist_enabled  = false
}
