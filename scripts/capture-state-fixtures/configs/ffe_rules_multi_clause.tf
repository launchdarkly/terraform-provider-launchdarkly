# Synthetic capture config for launchdarkly_feature_flag_environment
# with a rule carrying multiple clauses (multi-op + heterogeneous
# value_types).

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
  key  = "fixture-ffe-multi-clause-pj"
  name = "Phase 4 FFE multi-clause project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_feature_flag" "flag" {
  project_key    = launchdarkly_project.test.key
  key            = "fixture-multi-clause-flag"
  name           = "Phase 4 FFE multi-clause flag fixture"
  variation_type = "boolean"
  variations {
    value = "true"
  }
  variations {
    value = "false"
  }
}

resource "launchdarkly_feature_flag_environment" "multi_clause" {
  flag_id       = launchdarkly_feature_flag.flag.id
  env_key       = "fixture-env-test"
  on            = true
  off_variation = 1
  rules {
    variation = 0
    clauses {
      attribute = "age"
      op        = "greaterThan"
      values    = ["18"]
      value_type = "number"
    }
    clauses {
      attribute = "country"
      op        = "in"
      values    = ["US", "CA"]
    }
  }
  fallthrough {
    variation = 1
  }
}
