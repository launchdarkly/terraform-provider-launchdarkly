resource "launchdarkly_environment_release_plan" "checkout_rollout" {
  project_key     = launchdarkly_project.example.key
  environment_key = "production"
  key             = "checkout-rollout"
  name            = "Checkout rollout"
  description     = "Standard rollout for checkout changes"

  progressive_release = {
    randomization_unit = "user"
    advancement_mode   = "manual"
    cleanup_behavior   = "remove_rules_and_update_default_rule"

    phases = [
      {
        description = "Internal users"
        steps = [
          {
            allocation      = 10000
            duration_millis = 3600000
          },
        ]
        clauses = [
          {
            attribute    = "kind"
            op           = "in"
            values       = ["user"]
            context_kind = "user"
          },
          {
            attribute    = "email"
            op           = "endsWith"
            values       = ["@example.com"]
            context_kind = "user"
          },
        ]
      },
      {
        description = "Everyone"
        steps = [
          {
            allocation      = 25000
            duration_millis = 3600000
          },
          {
            allocation      = 50000
            duration_millis = 3600000
          },
          {
            allocation      = 100000
            duration_millis = 3600000
          },
        ]
        clauses = [
          {
            attribute    = "kind"
            op           = "in"
            values       = ["user"]
            context_kind = "user"
          },
        ]
      },
    ]

    metrics = {
      (launchdarkly_metric.completed_purchase.key) = {
        is_group      = false
        auto_rollback = true
      }
    }
  }
}
