# This config spins up a sample project to attach feature flags to
# since feature flags require association with a specific project

provider "launchdarkly" {
  version = ">= 2.0.0"
}

# since all projects are automatically created with a "test" and "production" 
# environment, no need to configure any envs
resource "launchdarkly_project" "tf_flag_examples" {
  key  = "tf-flag-examples"
  name = "Terraform Project for Flag Examples"

  # v2 of the LaunchDarkly Terraform provider requires you
  # define environments as part of your project resource configurations
  environments {
    name = "example environment"
    key = "example-env"
    color = "ababab"
    # You can configure approval settings per environment to control who can apply flag changes
    # Your Terraform user can be configured with a custom role to allow it to bypass approval requirements
    # See https://docs.launchdarkly.com/home/feature-workflows/environment-approvals#configuring-approval-settings
    approval_settings {
      min_num_approvals = 2
      required = true
    }
  }

  tags = [
    "terraform-managed",
  ]
}