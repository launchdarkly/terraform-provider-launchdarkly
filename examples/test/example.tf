terraform {
  required_providers {
    launchdarkly = {
      version = "~> 2.1.1"
      source  = "launchdarkly/launchdarkly"
    }
  }
  required_version = ">= 0.14"
}

data "launchdarkly_team_member" "test_person" {
  email = "ffeldberg+terraform@launchdarkly.com"
}
