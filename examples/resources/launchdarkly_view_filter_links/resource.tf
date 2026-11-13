# Each view these links target must exist before Terraform creates the links.
# Declare them (or read them with a launchdarkly_view data source when another
# configuration owns them) and reference view_key rather than repeating the key
# as a string literal. That reference is what tells Terraform to create the view
# first, and it rules out typos.
resource "launchdarkly_view" "frontend_team" {
  project_key         = "my-project"
  key                 = "frontend-team"
  name                = "Frontend Team"
  maintainer_team_key = "frontend"
}

resource "launchdarkly_view" "platform_team" {
  project_key         = "my-project"
  key                 = "platform-team"
  name                = "Platform Team"
  maintainer_team_key = "platform"
}

resource "launchdarkly_view" "beta_program" {
  project_key         = "my-project"
  key                 = "beta-program"
  name                = "Beta Program"
  maintainer_team_key = "product"
}

# Link all flags tagged "frontend" to a view
resource "launchdarkly_view_filter_links" "frontend_flags" {
  project_key = "my-project"
  view_key    = launchdarkly_view.frontend_team.key
  flag_filter = "tags:frontend"
}

# Link both flags and segments matching a tag
resource "launchdarkly_view_filter_links" "platform_resources" {
  project_key                   = "my-project"
  view_key                      = launchdarkly_view.platform_team.key
  flag_filter                   = "tags:platform"
  segment_filter                = "tags anyOf [\"platform\"]"
  segment_filter_environment_id = launchdarkly_project.my_project.environments["production"].client_side_id
}

# Link only segments matching a filter
resource "launchdarkly_view_filter_links" "beta_segments" {
  project_key                   = "my-project"
  view_key                      = launchdarkly_view.beta_program.key
  segment_filter                = "tags anyOf [\"beta\"]"
  segment_filter_environment_id = launchdarkly_project.my_project.environments["production"].client_side_id
}
