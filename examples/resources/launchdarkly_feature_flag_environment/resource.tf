resource "launchdarkly_feature_flag_environment" "number_env" {
  flag_id = launchdarkly_feature_flag.number.id
  env_key = launchdarkly_environment.staging.key

  on = true

  prerequisites {
    flag_key  = launchdarkly_feature_flag.basic.key
    variation = 0
  }

  targets {
    values    = ["user0"]
    variation = 0
  }
  targets {
    values    = ["user1", "user2"]
    variation = 1
  }
  context_targets {
    values       = ["accountX"]
    variation    = 1
    context_kind = "account"
  }

  rules {
    description = "example targeting rule with two clauses"
    clauses {
      attribute = "country"
      op        = "startsWith"
      values    = ["aus", "de", "united"]
      negate    = false
    }
    clauses {
      attribute = "segmentMatch"
      op        = "segmentMatch"
      values    = [launchdarkly_segment.example.key]
      negate    = false
    }
    variation = 0
  }

  fallthrough {
    rollout_weights = [60000, 40000, 0]
    context_kind    = "account"
    bucket_by       = "accountId"
  }
  off_variation = 2
}
