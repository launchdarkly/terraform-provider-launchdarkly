resource "launchdarkly_release_policy" "guarded_checkout" {
  project_key    = launchdarkly_project.example.key
  key            = "guarded-checkout"
  name           = "Guarded checkout rollout"
  release_method = "guarded-release"

  scope = {
    environment_keys = ["production"]
    flag_tag_keys    = ["checkout"]
  }

  guarded_release_config = {
    rollout_context_kind   = "user"
    min_sample_size        = 1000
    rollback_on_regression = true
    metric_keys            = [launchdarkly_metric.completed_purchase.key]

    stages = [
      {
        allocation      = 10
        duration_millis = 3600000
      },
      {
        allocation      = 50
        duration_millis = 3600000
      },
    ]
  }
}

resource "launchdarkly_release_policy" "progressive_rollout" {
  project_key    = launchdarkly_project.example.key
  key            = "progressive-rollout"
  name           = "Progressive rollout"
  release_method = "progressive-release"

  scope = {
    environment_keys = ["production"]
  }

  progressive_release_config = {
    rollout_context_kind = "user"

    stages = [
      {
        allocation      = 20
        duration_millis = 3600000
      },
      {
        allocation      = 60
        duration_millis = 3600000
      },
    ]
  }
}
