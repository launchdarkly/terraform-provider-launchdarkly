data "launchdarkly_environment_release_plan" "checkout_rollout" {
  project_key     = "example-project"
  environment_key = "production"
  key             = "checkout-rollout"
}
