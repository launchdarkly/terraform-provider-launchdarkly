data "launchdarkly_release_pipeline" "checkout_rollout" {
  project_key = "example-project"
  key         = "checkout-rollout"
}
