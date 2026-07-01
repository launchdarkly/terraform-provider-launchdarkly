# LaunchDarkly experiments can be imported using the experiment's ID in the form `project_key/environment_key/experiment_key`.
# Note: the iteration configuration is not populated on import and must be added to your configuration.
terraform import launchdarkly_experiment.checkout_button example-project/production/checkout-button-experiment
