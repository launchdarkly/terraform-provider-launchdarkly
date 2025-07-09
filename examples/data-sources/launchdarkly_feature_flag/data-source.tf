# Example: Get feature flag information including linked views
data "launchdarkly_feature_flag" "example" {
  project_key = "example-project"
  key         = "example-flag"
}

# The feature flag data source now includes discovery of linked views
output "flag_details" {
  value = {
    name        = data.launchdarkly_feature_flag.example.name
    description = data.launchdarkly_feature_flag.example.description
    views       = data.launchdarkly_feature_flag.example.views
    archived    = data.launchdarkly_feature_flag.example.archived
  }
}

# Example: Check if flag is accessible to specific teams
locals {
  accessible_to_frontend = contains(data.launchdarkly_feature_flag.example.views, "frontend-team")
  accessible_to_mobile   = contains(data.launchdarkly_feature_flag.example.views, "mobile-team")
  is_shared_flag         = length(data.launchdarkly_feature_flag.example.views) > 1
}

# Example: Conditional resource creation based on views
resource "launchdarkly_feature_flag_environment" "prod_config" {
  # Only create production config if flag is assigned to production view
  count = contains(data.launchdarkly_feature_flag.example.views, "production-ready") ? 1 : 0

  flag_id = data.launchdarkly_feature_flag.example.id
  env_key = "production"
  on      = true
  fallthrough {
    variation = 0
  }
  off_variation = 1
}

# Example: Generate team notifications based on views
output "team_notifications" {
  description = "Teams that should be notified about this flag"
  value = {
    flag_name = data.launchdarkly_feature_flag.example.name
    teams = [
      for view in data.launchdarkly_feature_flag.example.views : view
      if can(regex("-team$", view))
    ]
  }
}
