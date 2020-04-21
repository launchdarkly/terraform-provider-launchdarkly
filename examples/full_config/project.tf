# set up a project
provider "launchdarkly" {
  version = "~> 1.0"
}

resource "launchdarkly_project" "tf_full_config" {
  key  = "tf-full-config"
  name = "Terraform Full Example Configuration"

  tags = [
    "terraform"
  ]
}