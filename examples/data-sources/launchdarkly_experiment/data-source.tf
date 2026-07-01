data "launchdarkly_experiment" "checkout_button" {
  project_key     = "example-project"
  environment_key = "production"
  key             = "checkout-button-experiment"
}
