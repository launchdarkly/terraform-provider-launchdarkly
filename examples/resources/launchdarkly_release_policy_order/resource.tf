terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "~> 2.0"
    }
  }
}

# Configure the LaunchDarkly provider
provider "launchdarkly" {
  access_token = var.launchdarkly_access_token
}

# Create a project
resource "launchdarkly_project" "example" {
  key  = "example-project"
  name = "Example Project"

  environments {
    name  = "Test Environment"
    key   = "test"
    color = "000000"
  }

  environments {
    name  = "Production Environment"
    key   = "production"
    color = "ff0000"
  }
}

# Create multiple release policies
resource "launchdarkly_release_policy" "policy_a" {
  project_key    = launchdarkly_project.example.key
  key            = "policy-a"
  name           = "Policy A"
  release_method = "guarded-release"

  scope {
    environment_keys = ["production"]
  }

  guarded_release_config {
    rollback_on_regression = true
    min_sample_size        = 100
  }
}

resource "launchdarkly_release_policy" "policy_b" {
  project_key    = launchdarkly_project.example.key
  key            = "policy-b"
  name           = "Policy B"
  release_method = "progressive-release"

  scope {
    environment_keys = ["test"]
  }
}

resource "launchdarkly_release_policy" "policy_c" {
  project_key    = launchdarkly_project.example.key
  key            = "policy-c"
  name           = "Policy C"
  release_method = "guarded-release"

  scope {
    environment_keys = ["development"]
  }

  guarded_release_config {
    rollback_on_regression = false
    min_sample_size        = 50
  }
}

# Define the order of release policies within the project
resource "launchdarkly_release_policy_order" "example" {
  project_key = launchdarkly_project.example.key

  release_policy_keys = [
    launchdarkly_release_policy.policy_c.key,
    launchdarkly_release_policy.policy_a.key,
    launchdarkly_release_policy.policy_b.key
  ]
}

variable "launchdarkly_access_token" {
  description = "LaunchDarkly access token"
  type        = string
  sensitive   = true
}
