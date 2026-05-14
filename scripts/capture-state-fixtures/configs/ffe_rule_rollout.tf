# Synthetic capture config for launchdarkly_feature_flag_environment
# with a rule carrying rollout_weights (rollout path through
# rule_helper.go).

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
  key  = "fixture-ffe-rollout-pj"
  name = "Phase 4 FFE rollout project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_feature_flag" "flag" {
  project_key    = launchdarkly_project.test.key
  key            = "fixture-rollout-flag"
  name           = "Phase 4 FFE rollout flag fixture"
  variation_type = "boolean"
  variations {
    value = "true"
  }
  variations {
    value = "false"
  }
}

resource "launchdarkly_feature_flag_environment" "rollout" {
  flag_id       = launchdarkly_feature_flag.flag.id
  env_key       = "fixture-env-test"
  on            = true
  off_variation = 1
  rules {
    # v2.29 SDKv2 inflates variation=0 in state when omitted; setting
    # it explicitly avoids the plan-apply consistency check trap during
    # capture (state-compat capture playbook gotcha).
    variation       = 0
    rollout_weights = [60000, 40000]
    bucket_by       = "email"
    clauses {
      attribute = "country"
      op        = "in"
      values    = ["US"]
    }
  }
  fallthrough {
    variation = 1
  }
}
