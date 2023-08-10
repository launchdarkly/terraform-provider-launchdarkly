terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "~> 2.0.0"
    }
  }
  required_version = ">= 0.13"
}