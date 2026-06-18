resource "launchdarkly_release_pipeline" "checkout_rollout" {
  project_key = launchdarkly_project.example.key
  key         = "checkout-rollout"
  name        = "Checkout rollout"
  description = "Roll out checkout changes from internal testing to general availability"

  phases = [
    {
      name = "Internal testing"
      audiences = [
        {
          environment_key = "staging"
          name            = "QA team"
        },
      ]
    },
    {
      name = "General availability"
      audiences = [
        {
          environment_key = "production"
          name            = "All customers"
          configuration = {
            release_strategy = "manual"
            require_approval = true
            notify_team_keys = ["release-managers"]
          }
        },
      ]
    },
  ]

  tags = ["checkout", "terraform-managed"]

  depends_on = [launchdarkly_project.example]
}
