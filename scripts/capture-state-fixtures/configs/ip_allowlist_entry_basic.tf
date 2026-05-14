# Synthetic capture config for launchdarkly_ip_allowlist_entry
# (Phase 3.9). Uses a TEST-NET-1 reserved range (RFC 5737) so the
# entry can't affect access to the test LD account even briefly
# during the capture pass.

terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "2.29.0"
    }
  }
}

provider "launchdarkly" {}

resource "launchdarkly_ip_allowlist_entry" "basic" {
  ip_address  = "192.0.2.0/24"
  description = "Synthetic fixture entry (TEST-NET-1 RFC 5737)."
}
