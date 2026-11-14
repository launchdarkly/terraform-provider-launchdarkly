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

resource "launchdarkly_view" "mobile_team" {
  project_key         = "my-project"
  key                 = "mobile-team"
  name                = "Mobile Team"
  maintainer_team_key = "mobile"
}

resource "launchdarkly_view" "shared_features" {
  project_key         = "my-project"
  key                 = "shared-features"
  name                = "Shared Features"
  maintainer_team_key = "platform"
}

resource "launchdarkly_view" "backend_team" {
  project_key         = "my-project"
  key                 = "backend-team"
  name                = "Backend Team"
  maintainer_team_key = "backend"
}

resource "launchdarkly_view" "user_segments" {
  project_key         = "my-project"
  key                 = "user-segments-view"
  name                = "User Segments"
  maintainer_team_key = "platform"
}

# Example: Frontend team view with bulk flag and segment assignments
resource "launchdarkly_view_links" "frontend_team" {
  project_key = "my-project"
  view_key    = launchdarkly_view.frontend_team.key

  # Bulk link multiple flags efficiently - supports 100s of flags
  flags = [
    "feature-login",
    "feature-dashboard",
    "feature-payments",
    "feature-checkout",
    "feature-profile",
    "feature-notifications",
    "feature-search",
    "feature-filters",
    "feature-analytics",
    "feature-dark-mode",
    # Add more flag keys here to scale beyond 100 flags
  ]

  # Link segments relevant to this team's view
  segments = [
    {
      environment_id = "507f1f77bcf86cd799439011"
      segment_key    = "frontend-beta-users"
    },
    {
      environment_id = "507f1f77bcf86cd799439011"
      segment_key    = "premium-customers"
    },
  ]
}

# Example: Mobile team view with different flags
resource "launchdarkly_view_links" "mobile_team" {
  project_key = "my-project"
  view_key    = launchdarkly_view.mobile_team.key

  flags = [
    "feature-mobile-login",
    "feature-push-notifications",
    "feature-offline-mode",
    "feature-biometric-auth",
    "feature-mobile-payments",
    "feature-app-rating",
  ]

}

# Example: Shared features across teams
resource "launchdarkly_view_links" "shared_features" {
  project_key = "my-project"
  view_key    = launchdarkly_view.shared_features.key

  flags = [
    "feature-maintenance-mode",
    "feature-emergency-banner",
    "feature-api-throttling",
    "feature-logging-level",
  ]

}

# Demonstrating updates - adding/removing flags and segments from a view
resource "launchdarkly_view_links" "backend_team" {
  project_key = "my-project"
  view_key    = launchdarkly_view.backend_team.key

  flags = [
    "feature-database-migration",
    "feature-cache-optimization",
    "feature-api-versioning",
    # To add a new flag, add it to this list
    # To remove a flag, remove it from this list
    # Terraform will handle the link/unlink operations automatically
  ]

  # Link backend-specific segments across multiple environments
  segments = [
    {
      environment_id = "507f1f77bcf86cd799439011"
      segment_key    = "high-volume-api-users"
    },
    {
      environment_id = "507f1f77bcf86cd799439022" # Production environment
      segment_key    = "database-migration-pilot"
    },
  ]
}

# Example: View with only segments (no flags)
resource "launchdarkly_view_links" "segments_only" {
  project_key = "my-project"
  view_key    = launchdarkly_view.user_segments.key

  segments = [
    {
      environment_id = "507f1f77bcf86cd799439011"
      segment_key    = "vip-customers"
    },
    {
      environment_id = "507f1f77bcf86cd799439011"
      segment_key    = "enterprise-customers"
    },
    {
      environment_id = "507f1f77bcf86cd799439011"
      segment_key    = "trial-users"
    },
  ]
}
