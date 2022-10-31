terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = ">= 2.9.4"
    }
  }
  required_version = ">= 0.13"
}
