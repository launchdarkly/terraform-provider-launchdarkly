# set up a project
provider "launchdarkly" {
  version = ">= 1.0"
}

resource "launchdarkly_project" "tf_full_config" {
  key  = "tf-full-config-v2"
  name = "Terraform Full Example Configuration"
  # configure a production environment
  environments {
    name  = "Terraform Production Environment"
    key   = "production"
    color = "581845"
    tags = [
      "terraform"
    ]
  }
  # configure a staging environment
  environments {
    name  = "Terraform Staging Environment"
    key   = "staging"
    color = "FA9E8A"
    tags = [
      "terraform"
    ]
  }

  tags = [
    "terraform"
  ]
}
