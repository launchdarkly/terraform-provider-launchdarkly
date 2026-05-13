# Synthetic capture config for launchdarkly_audit_log_subscription
# (Phase 3.2). Uses the slack integration because its only required
# config field is `url`, which is a known integration_framework
# extra (no manifest entry, hard-coded in audit_log_subscription_helper.go).
# Datadog / msteams need real-looking webhook URLs that trip the
# fixture-safety scan; capture those separately.

terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "2.29.0"
    }
  }
}

provider "launchdarkly" {}

resource "launchdarkly_audit_log_subscription" "basic" {
  integration_key = "slack"
  name            = "fixture-audit-log-slack"
  on              = false

  config = {
    url = "https://example.invalid/fixture-token-PLACEHOLDER"
  }

  statements {
    actions   = ["*"]
    effect    = "allow"
    resources = ["proj/*:env/*:flag/*"]
  }

  tags = ["fixture"]
}
