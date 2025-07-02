# Example: Get view information including linked flags
data "launchdarkly_view" "example" {
  project_key = "example-project"
  key         = "example-view"
}

# The view data source now includes discovery of linked flags
output "view_details" {
  value = {
    name         = data.launchdarkly_view.example.name
    description  = data.launchdarkly_view.example.description
    linked_flags = data.launchdarkly_view.example.linked_flags
    maintainer   = data.launchdarkly_view.example.maintainer_id
  }
}

# Example: Use linked flags in conditional logic
locals {
  frontend_flags = contains(data.launchdarkly_view.example.linked_flags, "feature-frontend-redesign")
  has_experimental_flags = length([
    for flag in data.launchdarkly_view.example.linked_flags : flag
    if can(regex("^experimental-", flag))
  ]) > 0
}

# Example: Discover view membership for team access
data "launchdarkly_view" "team_view" {
  project_key = "my-project"
  key         = "frontend-team"
}

output "team_flag_access" {
  description = "Flags accessible to the frontend team"
  value = {
    view_name = data.launchdarkly_view.team_view.name
    flag_count = length(data.launchdarkly_view.team_view.linked_flags)
    flags = data.launchdarkly_view.team_view.linked_flags
  }
}
