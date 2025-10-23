resource "launchdarkly_feature_flag" "building_materials" {
  project_key = launchdarkly_project.example.key
  key         = "building-materials"
  name        = "Building materials"
  description = "this is a multivariate flag with string variations."

  variation_type = "string"
  variations {
    value       = "straw"
    name        = "Straw"
    description = "Watch out for wind."
  }
  variations {
    value       = "sticks"
    name        = "Sticks"
    description = "Sturdier than straw"
  }
  variations {
    value       = "bricks"
    name        = "Bricks"
    description = "The strongest variation"
  }

  client_side_availability {
    using_environment_id = false
    using_mobile_key     = true
  }

  defaults {
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
  variations {
    name  = "Single foo"
    value = jsonencode({ "foo" : "bar" })
  }
  variations {
    name  = "Multiple foos"
    value = jsonencode({ "foos" : ["bar1", "bar2"] })
  }

  defaults {
    on_variation  = 1
    off_variation = 0
  }
}

# Example: Feature flag with view associations
# This approach is ideal for modular Terraform where each flag is managed in its own file
resource "launchdarkly_feature_flag" "checkout_flow" {
  project_key = "example-project"
  key         = "checkout-flow-redesign"
  name        = "Checkout Flow Redesign"
  description = "New checkout experience with improved UX"

  variation_type = "boolean"

  # Link this flag to specific views
  # The flag will appear in both the "payments-team" and "frontend-team" views
  view_keys = [
    "payments-team",
    "frontend-team"
  ]

  tags = ["checkout", "payments", "frontend"]
}

# Example: Flag managed in a module that can specify its own views
# This enables a modular structure where each team/domain can manage their flags
# without needing to coordinate with a central view_links resource
resource "launchdarkly_feature_flag" "mobile_app_feature" {
  project_key = "example-project"
  key         = "mobile-push-notifications"
  name        = "Mobile Push Notifications"

  variation_type = "boolean"

  # Each flag can independently specify which views it belongs to
  view_keys = ["mobile-team"]

  tags = ["mobile", "notifications"]
}
