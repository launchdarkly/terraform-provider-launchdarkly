terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "~> 2.0"
    }
  }
  required_version = ">= 0.13"
}

resource "launchdarkly_project" "trigger_test" {
  key  = "trigger-test"
  name = "A Trigger Test Project"
  # configure a production environment
  environments {
    name  = "Terraform Production Environment"
    key   = "production"
    color = "581845"
  }
}

resource "launchdarkly_feature_flag" "trigger_test_flag" {
  project_key = launchdarkly_project.trigger_test.key
  key         = "trigger-test-flag"
  name        = "Trigger Test Flag"

  variation_type = "boolean"
}

resource "launchdarkly_flag_trigger" "test_trigger" {
  project_key     = launchdarkly_project.trigger_test.key
  env_key         = launchdarkly_project.trigger_test.environments.0.key
  flag_key        = launchdarkly_feature_flag.trigger_test_flag.key
  integration_key = "generic-trigger"
  instructions {
    kind = "turnFlagOff"
  }
  enabled = false
}
