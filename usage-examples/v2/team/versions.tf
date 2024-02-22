terraform {
  required_providers {
    launchdarkly = {
      source = "launchdarkly/launchdarkly"
    }
  }
  required_version = ">= 0.13"
}

provider "launchdarkly" {
  access_token = var.launchdarkly_access_token
}

