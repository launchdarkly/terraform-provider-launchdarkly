terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "2.29.0"
    }
  }
}

resource "launchdarkly_access_token" "fixture" {
  name = "fixture-access-token-basic"
  role = "reader"
}
