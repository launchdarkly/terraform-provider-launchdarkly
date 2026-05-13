terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "2.29.0"
    }
  }
}

resource "launchdarkly_webhook" "fixture" {
  url  = "https://example.invalid/fixture-webhook"
  name = "fixture-webhook-basic"
  on   = true
  tags = ["fixture-tag"]
  statements {
    effect    = "allow"
    resources = ["proj/*"]
    actions   = ["*"]
  }
}
