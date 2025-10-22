# Root module - Create the project and views

terraform {
  required_providers {
    launchdarkly = {
      source = "launchdarkly/launchdarkly"
    }
  }
}

provider "launchdarkly" {
  # API key configured via LAUNCHDARKLY_ACCESS_TOKEN environment variable
}

# Create the project
resource "launchdarkly_project" "example" {
  key  = "example-project"
  name = "Example Project"
}

# Create environments
resource "launchdarkly_environment" "production" {
  key        = "production"
  name       = "Production"
  project_key = launchdarkly_project.example.key
  color      = "FF0000"
}

resource "launchdarkly_environment" "staging" {
  key        = "staging"
  name       = "Staging"
  project_key = launchdarkly_project.example.key
  color      = "0000FF"
}

# Create views for each team
resource "launchdarkly_view" "payments_team" {
  project_key = launchdarkly_project.example.key
  key         = "payments-team"
  name        = "Payments Team"
  description = "View for the payments team"
  tags        = ["team"]
}

resource "launchdarkly_view" "frontend_team" {
  project_key = launchdarkly_project.example.key
  key         = "frontend-team"
  name        = "Frontend Team"
  description = "View for the frontend team"
  tags        = ["team"]
}

resource "launchdarkly_view" "shared_features" {
  project_key = launchdarkly_project.example.key
  key         = "shared-features"
  name        = "Shared Features"
  description = "View for cross-team shared features"
  tags        = ["shared"]
}

# Include team modules - each module defines its own flags with view_keys
module "payments" {
  source = "./modules/payments"
  
  project_key = launchdarkly_project.example.key
  
  # Pass view keys to module
  team_view_key   = launchdarkly_view.payments_team.key
  shared_view_key = launchdarkly_view.shared_features.key
  
  # Module handles flag creation and view linking internally
  depends_on = [
    launchdarkly_view.payments_team,
    launchdarkly_view.shared_features
  ]
}

module "frontend" {
  source = "./modules/frontend"
  
  project_key = launchdarkly_project.example.key
  
  # Pass view keys to module
  team_view_key   = launchdarkly_view.frontend_team.key
  shared_view_key = launchdarkly_view.shared_features.key
  
  # Module handles flag creation and view linking internally
  depends_on = [
    launchdarkly_view.frontend_team,
    launchdarkly_view.shared_features
  ]
}

