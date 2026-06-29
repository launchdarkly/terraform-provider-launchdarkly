resource "launchdarkly_experiment" "checkout_button" {
  project_key     = launchdarkly_project.example.key
  environment_key = "production"
  key             = "checkout-button-experiment"
  name            = "Checkout button experiment"
  description     = "Compare conversion rates between the control and treatment checkout buttons."
  tags            = ["experimentation"]

  iteration = {
    hypothesis                = "The green checkout button increases conversions."
    randomization_unit        = "user"
    primary_single_metric_key = launchdarkly_metric.checkout_conversion.key

    metrics = [{
      key = launchdarkly_metric.checkout_conversion.key
    }]

    treatments = [
      {
        name               = "Control"
        baseline           = true
        allocation_percent = "50"
        parameters = [{
          flag_key     = launchdarkly_feature_flag.checkout_button.key
          variation_id = "control-variation-id"
        }]
      },
      {
        name               = "Treatment"
        baseline           = false
        allocation_percent = "50"
        parameters = [{
          flag_key     = launchdarkly_feature_flag.checkout_button.key
          variation_id = "treatment-variation-id"
        }]
      },
    ]

    flags = {
      (launchdarkly_feature_flag.checkout_button.key) = {
        rule_id             = "fallthrough"
        flag_config_version = 1
      }
    }
  }
}
