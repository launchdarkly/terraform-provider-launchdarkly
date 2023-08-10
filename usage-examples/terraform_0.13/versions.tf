terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "~> 1.5"
    }
  }
  required_version = ">= 0.13"
}