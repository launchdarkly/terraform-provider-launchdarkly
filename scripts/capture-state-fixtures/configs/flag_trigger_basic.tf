terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "2.29.0"
    }
  }
}

resource "launchdarkly_project" "fixture" {
  key  = "fixture-project-1"
  name = "fixture-project-1"

  environments {
    key   = "fixture-env-test"
    name  = "fixture-env-test"
    color = "AABBCC"
  }
}

resource "launchdarkly_feature_flag" "fixture" {
  project_key = launchdarkly_project.fixture.key
  key         = "fixture-flag-1"
  name        = "fixture-flag-1"
  variation_type = "boolean"
}

resource "launchdarkly_flag_trigger" "fixture" {
  project_key      = launchdarkly_project.fixture.key
  env_key          = "fixture-env-test"
  flag_key         = launchdarkly_feature_flag.fixture.key
  integration_key  = "generic-trigger"
  enabled          = true
  instructions {
    kind = "turnFlagOn"
  }
}
