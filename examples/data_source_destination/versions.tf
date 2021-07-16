terraform {
  required_providers {
    launchdarkly = {
      source = "launchdarkly/launchdarkly"
      version = "~> 1.5.1"
    }
  }
  required_version = ">= 0.13"
}

# provider "launchdarkly" {
#   version = "~> 1.6.0"
#   source = "local/terraform-provider-launchdarkly"
# }