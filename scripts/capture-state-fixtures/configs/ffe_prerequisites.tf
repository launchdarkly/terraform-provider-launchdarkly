# Synthetic capture config for launchdarkly_feature_flag_environment
# with prerequisite flags.

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
  key  = "fixture-ffe-prereq-pj"
  name = "Phase 4 FFE prereq project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_feature_flag" "prereq" {
  project_key    = launchdarkly_project.test.key
  key            = "fixture-prereq-source"
  name           = "Prereq source"
  variation_type = "boolean"
  variations {
    value = "true"
  }
  variations {
    value = "false"
  }
}

resource "launchdarkly_feature_flag" "dependent" {
  project_key    = launchdarkly_project.test.key
  key            = "fixture-prereq-dep"
  name           = "Prereq dependent"
  variation_type = "boolean"
  variations {
    value = "true"
  }
  variations {
    value = "false"
  }
}

resource "launchdarkly_feature_flag_environment" "dependent" {
  flag_id       = launchdarkly_feature_flag.dependent.id
  env_key       = "fixture-env-test"
  on            = true
  off_variation = 1
  prerequisites {
    flag_key  = launchdarkly_feature_flag.prereq.key
    variation = 0
  }
  fallthrough {
    variation = 0
  }
}
