# This example shows the use of tags, targets, context targets, and rules for a segment
resource "launchdarkly_segment" "example" {
  key         = "example-segment-key"
  project_key = launchdarkly_project.example.key
  env_key     = launchdarkly_environment.example.key
  name        = "example segment"
  description = "This segment is managed by Terraform"
  tags        = ["segment-tag-1", "segment-tag-2"]
  included    = ["user1", "user2"]
  excluded    = ["user3", "user4"]
  included_contexts = [{
    values       = ["account1", "account2"]
    context_kind = "account"
  }]

  rules = [{
    clauses = [{
      attribute    = "country"
      op           = "startsWith"
      values       = ["en", "de", "un"]
      negate       = false
      context_kind = "location-data"
    }]
  }]
}

# This example shows a segment configured to have an unbounded number of individual targets
resource "launchdarkly_segment" "big-example" {
  key                    = "example-big-segment-key"
  project_key            = launchdarkly_project.example.key
  env_key                = launchdarkly_environment.example.key
  name                   = "example big segment"
  description            = "This big segment is managed by Terraform"
  tags                   = ["segment-tag-1", "segment-tag-2"]
  unbounded              = true
  unbounded_context_kind = "user"
}

# This example shows a segment with a targeting rule that uses all clause operators
resource "launchdarkly_segment" "segment_with_all_clause_operators" {
  name        = "Segment with all clause operators"
  key         = "segment-operators"
  project_key = "projectx"
  env_key     = "development"

  rules = [{
    clauses = [
      {
        attribute = "username"
        op        = "in" // Maps to 'is one of' in the UI
        values    = ["henrietta powell", "wally waterbear"]
      },
      {
        attribute = "username"
        op        = "endsWith" // Maps to 'ends with' in the UI
        values    = ["powell", "waterbear"]
      },
      {
        attribute = "username"
        op        = "startsWith" // Maps to 'starts with' in the UI
        values    = ["henrietta", "wally"]
      },
      {
        attribute = "username"
        op        = "matches" // Maps to 'matches regex' in the UI
        values    = ["henr*"]
      },
      {
        attribute = "username"
        op        = "contains" // Maps to 'contains' in the UI
        values    = ["water"]
      },
      {
        attribute = "pageVisits"
        op        = "lessThan" // Maps to 'less than (<)' in the UI
        values    = [100]
      },
      {
        attribute = "pageVisits"
        op        = "lessThanOrEqual" // Maps to 'less than or equal to (<=)' in the UI
        values    = [100]
      },
      {
        attribute = "pageVisits"
        op        = "greaterThan" // Maps to 'greater than (>)' in the UI
        values    = [100]
      },
      {
        attribute = "pageVisits"
        op        = "greaterThanOrEqual" // Maps to 'greater than or equal to (>=)' in the UI
        values    = [100]
      },
      {
        attribute = "creationDate"
        op        = "before" // Maps to 'before' in the UI
        values    = ["2024-05-03T15:57:30Z"]
      },
      {
        attribute = "creationDate"
        op        = "after" // Maps to 'after' in the UI
        values    = ["2024-05-03T15:57:30Z"]
      },
      {
        attribute    = "version"
        op           = "semVerEqual" // Maps to 'semantic version is one of (=)' in the UI
        values       = ["1.0.0", "1.0.1"]
        context_kind = "application"
      },
      {
        attribute    = "version"
        op           = "semVerLessThan" // Maps to 'semantic version less than (<)' in the UI
        values       = ["1.0.0"]
        context_kind = "application"
      },
      {
        attribute    = "version"
        op           = "semVerGreaterThan" // Maps to 'semantic version greater than (>)' in the UI
        values       = ["1.0.0"]
        context_kind = "application"
      },
      {
        attribute = "context"
        op        = "segmentMatch" // Maps to 'Context is in' in the UI
        values    = ["test-segment"]
      },
    ]
  }]
}

# Example: Segment with view associations
# This approach is ideal for modular Terraform where each segment is managed in its own file
#
# Always reference the view rather than repeating its key as a string literal.
# A view must exist before Terraform can link a segment to it, and Terraform only
# knows to create the view first if the segment references it. With a string
# literal there is no dependency between the two resources, so Terraform can
# create the segment first and the apply fails with "view does not exist".
# Referencing the view also rules out a mistyped key.
resource "launchdarkly_view" "sales_team" {
  project_key         = "example-project"
  key                 = "sales-team"
  name                = "Sales Team"
  maintainer_team_key = "sales"
}

resource "launchdarkly_view" "customer_success" {
  project_key         = "example-project"
  key                 = "customer-success"
  name                = "Customer Success"
  maintainer_team_key = "customer-success"
}

resource "launchdarkly_segment" "premium_users" {
  key         = "premium-users"
  project_key = "example-project"
  env_key     = "production"
  name        = "Premium Users"
  description = "Users with premium subscriptions"

  # The segment appears in both the "sales-team" and "customer-success" views.
  # Terraform creates both views first because of these references.
  view_keys = [
    launchdarkly_view.sales_team.key,
    launchdarkly_view.customer_success.key,
  ]

  tags = ["premium", "subscription"]

  rules = [{
    clauses = [{
      attribute = "plan"
      op        = "in"
      values    = ["premium", "enterprise"]
    }]
  }]
}

# Example: Segment managed in a module that can specify its own views
# This enables a modular structure where each team/domain can manage their segments
# without needing to coordinate with a central view_links resource
#
# When the view is owned by another configuration or state (for example a
# platform team's workspace), use the data source. This asserts the view already
# exists, so a typo or a missing view fails during plan instead of mid-apply.
data "launchdarkly_view" "product_team" {
  project_key = "example-project"
  key         = "product-team"
}

resource "launchdarkly_segment" "beta_testers" {
  key         = "beta-testers"
  project_key = "example-project"
  env_key     = "staging"
  name        = "Beta Testers"

  # Each segment can independently specify which views it belongs to
  view_keys = [data.launchdarkly_view.product_team.key]

  tags = ["beta", "testing"]

  included = ["user123", "user456"]
}

# Removing view associations
# - To remove all view associations, set view_keys = []
# - Removing the view_keys field from your configuration leaves existing associations unchanged
# - The field is computed, so Terraform detects drift if associations change outside Terraform
