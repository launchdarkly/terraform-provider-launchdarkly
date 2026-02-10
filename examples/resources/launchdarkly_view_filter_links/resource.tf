# Link all flags tagged "frontend" to a view
resource "launchdarkly_view_filter_links" "frontend_flags" {
  project_key = "my-project"
  view_key    = "frontend-team"
  flag_filter = "tags:frontend"
}

# Link both flags and segments matching a tag
resource "launchdarkly_view_filter_links" "platform_resources" {
  project_key    = "my-project"
  view_key       = "platform-team"
  flag_filter    = "tags:platform"
  segment_filter = "tags:platform"
}

# Link only segments matching a filter
resource "launchdarkly_view_filter_links" "beta_segments" {
  project_key    = "my-project"
  view_key       = "beta-program"
  segment_filter = "tags:beta"
}
