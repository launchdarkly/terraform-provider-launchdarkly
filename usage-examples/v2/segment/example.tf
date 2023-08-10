# This config provides an example for configuring a user segment

provider "launchdarkly" {
  version = "~> 2.0.0"
}

resource "launchdarkly_project" "example" {
  key  = "example-project"
  name = "segment example project"
  environments {
    name  = "example environment"
    key   = "example-env"
    color = "010101"
  }
}

resource "launchdarkly_segment" "example_segment" {
  key         = "example-segment"
  project_key = launchdarkly_project.example.key
  env_key     = launchdarkly_project.example.environments.0.key
  name        = "example segment"
  description = "This is an example segment managed by Terraform"
  tags        = ["terraform-managed", "example-tag"]
  included    = ["user1", "user2"]
  excluded    = ["user3", "user4"]

  rules {
    clauses {
      attribute = "country"
      op        = "startsWith"
      values    = ["en", "de", "un"]
      negate = true
    }
  }
}