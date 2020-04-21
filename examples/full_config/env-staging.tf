# create a staging environment
resource "launchdarkly_environment" "staging" {
  name  = "Terraform Staging Environment"
  key   = "staging"
  color = "FA9E8A"
  tags = [
    "terraform"
  ]

  project_key = launchdarkly_project.tf_full_config.key
}

# configure env-specific flag attributes on the ld_internal_tester flag in flags.tf
# requires the binary_flag to be on to apply
resource "launchdarkly_feature_flag_environment" "ld_internal_tester_staging" {
  flag_id           = launchdarkly_feature_flag.ld_internal_tester.id
  env_key           = "staging"
  targeting_enabled = true

  prerequisites {
    flag_key  = launchdarkly_feature_flag.binary_flag.key
    variation = 1
  }

  rules {
    clauses {
      attribute = "company"
      op        = "matches"
      values    = ["LaunchDarkly"]
      negate    = false
    }
  }
}