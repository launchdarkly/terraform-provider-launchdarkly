# This config provides examples for setting up complex targeting rules 
# with percent rollouts, prerequisites, and bucketing.

# This flag provides an example usage of the prerequisites feature:
# the number_flag will only be served where the boolean_flag is already on
# and in the specified countries.
resource "launchdarkly_feature_flag_environment" "prereq_flag" {
  flag_id           = launchdarkly_feature_flag.number_flag.id
  env_key           = "production"
  targeting_enabled = true

  prerequisites {
    flag_key  = launchdarkly_feature_flag.boolean_flag.key
    variation = 1
  }

  rules {
    clauses {
      attribute = "country"
      op        = "matches"
      values    = ["uk", "aus", "usa"]
      negate    = false
    }
  }
}

# This flag provides an example of user-specific targeting in the test environment
# on the string_flag defined in "flag_types_example.tf".
# The order of the user_targets blocks determines the index of the variation
# to be served to each set of users.
# The rules block of this resource determines that the 0-index variation ("string1") will
# be served to users whose names start with the letters a-e.
# flag_fallthrough describes the default to serve if none of the other rules apply:
# in this case, the percentage of users who will be served each variation (must sum to 100000).
# Use of the bucket_by attribute ensures that all users with the same company will be served the 
# same variation within the rollout buckets.
resource "launchdarkly_feature_flag_environment" "user_targeting_flag" {
  flag_id           = launchdarkly_feature_flag.string_flag.id
  env_key           = "test"
  targeting_enabled = true
  track_events      = true
  user_targets {
    values = ["test_user0"]
  }
  user_targets {
    values = ["test_user1", "test_user2"]
  }
  user_targets {
    values = ["test_user3"]
  }
  rules {
    clauses {
      attribute = "name"
      op        = "startsWith"
      values    = ["a", "b", "c", "d", "e"]
      negate    = false
    }
    variation = 0
  }

  flag_fallthrough {
    rollout_weights = [60000, 30000, 10000]
    bucket_by       = "company"
  }
}
