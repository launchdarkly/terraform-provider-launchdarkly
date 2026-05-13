terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "2.29.0"
    }
  }
}

resource "launchdarkly_relay_proxy_configuration" "fixture" {
  name = "fixture-relay-proxy-basic"
  policy {
    effect    = "allow"
    resources = ["proj/*:env/*"]
    actions   = ["*"]
  }
}
