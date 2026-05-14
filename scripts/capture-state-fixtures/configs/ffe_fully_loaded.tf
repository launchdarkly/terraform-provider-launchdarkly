# Synthetic capture config for launchdarkly_feature_flag_environment
# exercising every nested block (rules + clauses + targets +
# context_targets + prerequisites + fallthrough with rollout).

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
  key  = "fixture-ffe-fully-pj"
  name = "Phase 4 FFE fully-loaded project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_feature_flag" "prereq" {
  project_key    = launchdarkly_project.test.key
  key            = "fixture-fully-prereq"
  name           = "Fully-loaded prereq source"
  variation_type = "boolean"
  variations {
    value = "true"
  }
  variations {
    value = "false"
  }
}

resource "launchdarkly_feature_flag" "main" {
  project_key    = launchdarkly_project.test.key
  key            = "fixture-fully-flag"
  name           = "Fully-loaded flag"
  variation_type = "string"
  variations {
    value = "alpha"
  }
  variations {
    value = "beta"
  }
  variations {
    value = "gamma"
  }
}

resource "launchdarkly_feature_flag_environment" "fully" {
  flag_id       = launchdarkly_feature_flag.main.id
  env_key       = "fixture-env-test"
  on            = true
  off_variation = 2
  track_events  = false

  prerequisites {
    flag_key  = launchdarkly_feature_flag.prereq.key
    variation = 0
  }

  targets {
    values    = ["fixture-user-power"]
    variation = 0
  }

  context_targets {
    context_kind = "organization"
    values       = ["acme"]
    variation    = 1
  }

  rules {
    description = "geo-restricted"
    variation   = 0
    clauses {
      attribute = "country"
      op        = "in"
      values    = ["US"]
    }
  }

  rules {
    description = "rollout to beta"
    # v2.29 SDKv2 inflates variation=0 in state when omitted; setting
    # it explicitly avoids the plan-apply consistency check trap during
    # capture (state-compat capture playbook gotcha).
    variation       = 0
    rollout_weights = [50000, 25000, 25000]
    bucket_by       = "email"
    clauses {
      attribute = "tier"
      op        = "in"
      values    = ["pro", "enterprise"]
    }
  }

  fallthrough {
    rollout_weights = [33333, 33334, 33333]
  }
}
