# Example: Frontend team view with bulk flag assignments
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
  
  comment = "Frontend team flag assignments managed by Terraform"
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

# Demonstrating updates - adding/removing flags from a view
resource "launchdarkly_view_links" "backend_team" {
  project_key = "my-project"
  view_key    = "backend-team"
  
  flags = [
    "feature-database-migration",
    "feature-cache-optimization",
    "feature-api-versioning",
    # To add a new flag, simply add it to this list
    # To remove a flag, remove it from this list
    # Terraform will handle the link/unlink operations automatically
  ]
  
  comment = "Backend infrastructure and API flags"
}

# To import an existing view's flag links into Terraform:
# terraform import launchdarkly_view_links.frontend_team my-project/frontend-team 