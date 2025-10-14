# Example: Frontend team view with bulk flag and segment assignments
resource "launchdarkly_view_links" "frontend_team" {
  project_key = "my-project"
  view_key    = "frontend-team"

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
    # ... can easily scale to 100+ flags
  ]

  # Link segments relevant to this team's view
  segments {
    environment_id = "507f1f77bcf86cd799439011"
    segment_key    = "frontend-beta-users"
  }

  segments {
    environment_id = "507f1f77bcf86cd799439011"
    segment_key    = "premium-customers"
  }
}

# Example: Mobile team view with different flags
resource "launchdarkly_view_links" "mobile_team" {
  project_key = "my-project"
  view_key    = "mobile-team"

  flags = [
    "feature-mobile-login",
    "feature-push-notifications",
    "feature-offline-mode",
    "feature-biometric-auth",
    "feature-mobile-payments",
    "feature-app-rating",
  ]

  comment = "Mobile team specific features"
}

# Example: Shared features across teams
resource "launchdarkly_view_links" "shared_features" {
  project_key = "my-project"
  view_key    = "shared-features"

  flags = [
    "feature-maintenance-mode",
    "feature-emergency-banner",
    "feature-api-throttling",
    "feature-logging-level",
  ]

  comment = "Cross-team shared feature flags"
}

# Demonstrating updates - adding/removing flags and segments from a view
resource "launchdarkly_view_links" "backend_team" {
  project_key = "my-project"
  view_key    = "backend-team"

  flags = [
    "feature-database-migration",
    "feature-cache-optimization",
    "feature-api-versioning",
    # To add a new flag, add it to this list
    # To remove a flag, remove it from this list
    # Terraform will handle the link/unlink operations automatically
  ]

  # Link backend-specific segments across multiple environments
  segments {
    environment_id = "507f1f77bcf86cd799439011"
    segment_key    = "high-volume-api-users"
  }

  segments {
    environment_id = "507f1f77bcf86cd799439022" # Production environment
    segment_key    = "database-migration-pilot"
  }
}

# Example: View with only segments (no flags)
resource "launchdarkly_view_links" "segments_only" {
  project_key = "my-project"
  view_key    = "user-segments-view"

  segments {
    environment_id = "507f1f77bcf86cd799439011"
    segment_key    = "vip-customers"
  }

  segments {
    environment_id = "507f1f77bcf86cd799439011"
    segment_key    = "enterprise-customers"
  }

  segments {
    environment_id = "507f1f77bcf86cd799439011"
    segment_key    = "trial-users"
  }
}
