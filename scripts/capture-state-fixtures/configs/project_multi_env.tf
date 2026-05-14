# Synthetic capture config for launchdarkly_project with multiple
# environments. Tests the environment-list ordering parity through the
# nested environments block.

terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "2.29.0"
    }
  }
}

provider "launchdarkly" {}

resource "launchdarkly_project" "multi_env" {
  key  = "fixture-project-multi"
  name = "Phase 4 project multi-env fixture"
  tags = ["fixture", "multi-env"]

  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
  environments {
    key   = "fixture-env-staging"
    name  = "Fixture staging env"
    color = "445566"
  }
  environments {
    key   = "fixture-env-prod"
    name  = "Fixture prod env"
    color = "778899"
  }
}
