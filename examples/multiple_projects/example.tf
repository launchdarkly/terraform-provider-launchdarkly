# This file demonstrates the simultaneous configuration of multiple projects,
# each with their own environments and feature flags.

# ----------------------------------------------------------------------------------- #
# AUTH CONFIG
provider "launchdarkly" {
  version = "~> 1.0"
}

# ----------------------------------------------------------------------------------- #
# PROJECT 1

# create the project with nested environments blocks
resource "launchdarkly_project" "tf_project_1" {
  key  = "tf-project-1"
  name = "Terraform Example Project 1"

  tags = [
    "terraform-managed",
  ]

  # create a terraform-specific test environment within the project
  environments {
    name  = "Terraform Test Environment"
    key   = "tf-test"
    color = "999999"
    tags = [
      "terraform-managed",
      "test"
    ]
  }

  # create a terraform-specific production environment within the project
  environments {
    name  = "Terraform Production Environment"
    key   = "tf-production"
    color = "333333"
    tags = [
      "terraform-managed",
    ]
  }
}

# create a basic feature flag within project 1
resource "launchdarkly_feature_flag" "basic" {
  project_key    = launchdarkly_project.tf_project_1.key
  key            = "basic-flag"
  name           = "Basic feature flag"
  variation_type = "boolean"
}

# create flag attributes specific to the tf_ env on the basic flag defined above
# In this case, if the basic flag is set on variation 1,
# the flag will display variation 0 only to users whose country matches "de" or "fr".
resource "launchdarkly_feature_flag_environment" "basic_variation" {
  flag_id = launchdarkly_feature_flag.basic.id

  # since the environment was specified as a nested block in the project resource, the environment
  # key must be retrieved through the `launchdarkly_project` resource.
  env_key = launchdarkly_project.tf_project_1.environments.0.key

  targeting_enabled = true

  rules {
    clauses {
      attribute = "country"
      op        = "matches"
      values    = ["de", "fr"]
      negate    = false
    }
    variation = 0
  }

  flag_fallthrough {
    variation = 0
  }
}

# ----------------------------------------------------------------------------------- #
# PROJECT 2 - an example of how to use dedicated `launchdarkly_enviromnent` resources instead
# of nested environments in the `launchdarkly_project` resource 

resource "launchdarkly_project" "tf_project_2" {
  key  = "tf-project-2"
  name = "Terraform Example Project 2"

  tags = [
    "terraform-managed",
  ]
}

resource "launchdarkly_environment" "tf_env_a" {
  name  = "Example Environment A"
  key   = "tf-example-env-a"
  color = "ff00ff"
  tags = [
    "terraform-managed",
    "rollouts"
  ]

  project_key = launchdarkly_project.tf_project_2.key
}

resource "launchdarkly_environment" "tf_env_b" {
  name  = "Example Environment B"
  key   = "tf-example-env-b"
  color = "00FFFF"
  tags = [
    "terraform-managed",
  ]

  project_key = launchdarkly_project.tf_project_2.key
}
