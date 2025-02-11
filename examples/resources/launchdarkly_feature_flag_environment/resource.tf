# This example shows the use of prerequisites, targets, context targets, rules, and fallthrough for a feature flag environment
resource "launchdarkly_feature_flag_environment" "number_ff_env" {
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

# This example shows the minimum configuration required to create a feature flag environment
resource "launchdarkly_feature_flag_environment" "basic_flag_environment" {
  flag_id = launchdarkly_feature_flag.basic_flag.id
  env_key = "development"

  on = true

  fallthrough {
    variation = 1
  }
  off_variation = 0
}

# This example shows a feature flag environment with a targeting rule that uses every clause operator
resource "launchdarkly_feature_flag_environment" "big_flag_environment" {
  flag_id = launchdarkly_feature_flag.big_flag.id
  env_key = "development"

  on = true

  rules {
    description = "Example targeting rule with every clause operator"
    clauses {
      attribute = "username"
      op        = "in" // Maps to 'is one of' in the UI
      values    = ["henrietta powell", "wally waterbear"]
    }
    clauses {
      attribute = "username"
      op        = "endsWith" // Maps to 'ends with' in the UI
      values    = ["powell", "waterbear"]
    }
    clauses {
      attribute = "username"
      op        = "startsWith" // Maps to 'starts with' in the UI
      values    = ["henrietta", "wally"]
    }
    clauses {
      attribute = "username"
      op        = "matches" // Maps to 'matches regex' in the UI
      values    = ["henr*"]
    }
    clauses {
      attribute = "username"
      op        = "contains" // Maps to 'contains' in the UI
      values    = ["water"]
    }
    clauses {
      attribute = "pageVisits"
      op        = "lessThan" // Maps to 'less than (<)' in the UI
      values    = [100]
    }
    clauses {
      attribute = "pageVisits"
      op        = "lessThanOrEqual" // Maps to 'less than or equal to (<=)' in the UI
      values    = [100]
    }
    clauses {
      attribute = "pageVisits"
      op        = "greaterThan" // Maps to 'greater than (>)' in the UI
      values    = [100]
    }
    clauses {
      attribute = "pageVisits"
      op        = "greaterThanOrEqual" // Maps to 'greater than or equal to (>=)' in the UI
      values    = [100]
    }
    clauses {
      attribute = "creationDate"
      op        = "before" // Maps to 'before' in the UI
      values    = ["2024-05-03T15:57:30Z"]
    }
    clauses {
      attribute = "creationDate"
      op        = "after" // Maps to 'after' in the UI
      values    = ["2024-05-03T15:57:30Z"]
    }
    clauses {
      attribute    = "version"
      op           = "semVerEqual" // Maps to 'semantic version is one of (=)' in the UI
      values       = ["1.0.0", "1.0.1"]
      context_kind = "application"
    }
    clauses {
      attribute    = "version"
      op           = "semVerLessThan" // Maps to 'semantic version less than (<)' in the UI
      values       = ["1.0.0"]
      context_kind = "application"
    }
    clauses {
      attribute    = "version"
      op           = "semVerGreaterThan" // Maps to 'semantic version greater than (>)' in the UI
      values       = ["1.0.0"]
      context_kind = "application"
    }
    clauses {
      attribute = "context"
      op        = "segmentMatch" // Maps to 'Context is in' in the UI
      values    = ["test-segment"]
    }
    rollout_weights = [40000, 60000]
    bucket_by       = "country"
    context_kind    = "account"
  }

  fallthrough {
    variation = 1
  }
  off_variation = 0
}
