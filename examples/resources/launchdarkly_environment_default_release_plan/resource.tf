resource "launchdarkly_environment_release_plan" "checkout_rollout" {
  project_key     = launchdarkly_project.example.key
  environment_key = "production"
  key             = "checkout-rollout"
  name            = "Checkout rollout"

  progressive_release = {
    randomization_unit = "user"

    phases = [
      {
        steps = [
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
  }
}

resource "launchdarkly_environment_default_release_plan" "production" {
  project_key      = launchdarkly_project.example.key
  environment_key  = launchdarkly_environment_release_plan.checkout_rollout.environment_key
  release_plan_key = launchdarkly_environment_release_plan.checkout_rollout.key
}
