# Example: Get release policy information
data "launchdarkly_release_policy" "example" {
  project_key = "example-project"
  key         = "example-policy"
}

# Use the release policy information
output "policy_details" {
  value = {
    name           = data.launchdarkly_release_policy.example.name
    release_method = data.launchdarkly_release_policy.example.release_method
    environments   = data.launchdarkly_release_policy.example.scope[0].environment_keys
  }
}

# Example: Check if policy is guarded release
locals {
  is_guarded_release = data.launchdarkly_release_policy.example.release_method == "guarded-release"
  has_rollback       = length(data.launchdarkly_release_policy.example.guarded_release_config) > 0 ? data.launchdarkly_release_policy.example.guarded_release_config[0].rollback_on_regression : false
}

# Example: Conditional logic based on release policy
resource "launchdarkly_feature_flag" "conditional_flag" {
  project_key    = data.launchdarkly_release_policy.example.project_key
  key            = "my-feature-flag"
  name           = "My Feature Flag"
  variation_type = "boolean"

  # Use release policy configuration to set defaults
  defaults {
    on_variation  = local.is_guarded_release ? 0 : 1
    off_variation = 0
  }
}
