// Managing environments directly using the launchdarkly_environment resource is only
// recommended if you intend to manage the project outside of Terraform. If you wish to
// test this configuration, please update the project_key to match an existing project in
// your LaunchDarkly account. See the documentation for more information.
terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "~> 2.0"
    }
  }
  required_version = ">= 0.13"
}

resource "launchdarkly_environment" "env_test" {
  key             = "testing-bug"
  name            = "testing bug"
  color           = "AAAAAA"
  project_key     = "default"
  confirm_changes = "false"
}
