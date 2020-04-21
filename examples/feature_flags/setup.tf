# This config spins up a sample project to attach feature flags to
# since feature flags require association with a specific project

provider "launchdarkly" {
  version = ">= 1.2.0"
}

# since all projects are automatically created with a "test" and "production" 
# environment, no need to configure any envs
resource "launchdarkly_project" "tf_flag_examples" {
  key  = "tf-flag-examples"
  name = "Terraform Project for Flag Examples"

  tags = [
    "terraform-managed",
  ]
}