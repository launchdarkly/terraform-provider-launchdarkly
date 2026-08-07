resource "launchdarkly_feature_flag" "building_materials" {
  project_key = launchdarkly_project.example.key
  key         = "building-materials"
  name        = "Building materials"
  description = "this is a multivariate flag with string variations."

  variation_type = "string"
  variations = [
    {
      value       = "straw"
      name        = "Straw"
      description = "Watch out for wind."
    },
    {
      value       = "sticks"
      name        = "Sticks"
      description = "Sturdier than straw"
    },
    {
      value       = "bricks"
      name        = "Bricks"
      description = "The strongest variation"
    },
  ]

  client_side_availability = {
    using_environment_id = false
    using_mobile_key     = true
  }

  defaults = {
    on_variation  = 2
    off_variation = 0
  }

  tags = [
    "example",
    "terraform",
    "multivariate",
    "building-materials",
  ]
}

resource "launchdarkly_feature_flag" "json_example" {
  project_key = "example-project"
  key         = "json-example"
  name        = "JSON example flag"

  variation_type = "json"
  variations = [
    {
      name  = "Single foo"
      value = jsonencode({ "foo" : "bar" })
    },
    {
      name  = "Multiple foos"
      value = jsonencode({ "foos" : ["bar1", "bar2"] })
    },
  ]

  defaults = {
    on_variation  = 1
    off_variation = 0
  }
}

# Example: Feature flag with view associations
# This approach is ideal for modular Terraform where each flag is managed in its own file
#
# Always reference the view rather than repeating its key as a string literal.
# A view must exist before Terraform can link a flag to it, and Terraform only
# knows to create the view first if the flag references it. With a string literal
# there is no dependency between the two resources, so Terraform can create the
# flag first and the apply fails with "view does not exist". Referencing the view
# also rules out a mistyped key.
resource "launchdarkly_view" "payments_team" {
  project_key         = "example-project"
  key                 = "payments-team"
  name                = "Payments Team"
  maintainer_team_key = "payments"
}

resource "launchdarkly_view" "frontend_team" {
  project_key         = "example-project"
  key                 = "frontend-team"
  name                = "Frontend Team"
  maintainer_team_key = "frontend"
}

resource "launchdarkly_feature_flag" "checkout_flow" {
  project_key = "example-project"
  key         = "checkout-flow-redesign"
  name        = "Checkout Flow Redesign"
  description = "New checkout experience with improved UX"

  variation_type = "boolean"

  # The flag appears in both the "payments-team" and "frontend-team" views.
  # Terraform creates both views before this flag because of these references.
  view_keys = [
    launchdarkly_view.payments_team.key,
    launchdarkly_view.frontend_team.key,
  ]

  tags = ["checkout", "payments", "frontend"]
}

# Example: Flag managed in a module that can specify its own views
# This enables a modular structure where each team/domain can manage their flags
# without needing to coordinate with a central view_links resource
#
# When the view is owned by another configuration or state (for example a
# platform team's workspace), use the data source. This asserts the view already
# exists, so a typo or a missing view fails during plan instead of mid-apply.
data "launchdarkly_view" "mobile_team" {
  project_key = "example-project"
  key         = "mobile-team"
}

resource "launchdarkly_feature_flag" "mobile_app_feature" {
  project_key = "example-project"
  key         = "mobile-push-notifications"
  name        = "Mobile Push Notifications"

  variation_type = "boolean"

  # Each flag can independently specify which views it belongs to
  view_keys = [data.launchdarkly_view.mobile_team.key]

  tags = ["mobile", "notifications"]
}

# Removing view associations
# - To remove all view associations, set view_keys = []
# - Removing the view_keys field from your configuration leaves existing associations unchanged
# - The field is computed, so Terraform detects drift if associations change outside Terraform
