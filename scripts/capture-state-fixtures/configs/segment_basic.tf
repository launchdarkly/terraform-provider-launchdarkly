# Synthetic capture config for launchdarkly_segment (Phase 4.2):
# minimal segment with name + key + project + environment, no rules
# or targeting.

terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "2.29.0"
    }
  }
}

provider "launchdarkly" {}

resource "launchdarkly_project" "test" {
  key  = "fixture-segment-basic-pj"
  name = "Phase 4 segment basic project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_segment" "basic" {
  project_key = launchdarkly_project.test.key
  env_key     = "fixture-env-test"
  key         = "fixture-segment-basic"
  name        = "Phase 4 segment basic fixture"
}
