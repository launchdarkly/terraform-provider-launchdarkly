terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "~> 3.0"
    }
  }
}

# Configure the LaunchDarkly provider
provider "launchdarkly" {
  # The access token can also be set with the LAUNCHDARKLY_ACCESS_TOKEN environment variable.
  access_token = var.launchdarkly_access_token

  # Optional. The maximum number of concurrent API requests the provider makes. Defaults to 1.
  # Raise it to speed up plan and refresh on large configurations, at the cost of a higher chance
  # of hitting your account's API rate limit.
  max_concurrency = 1

  # Optional. When true, removing a launchdarkly_feature_flag from your configuration archives the
  # flag in LaunchDarkly instead of deleting it. Defaults to false.
  archive_flags_on_destroy = false
}

# Create a project with a single environment
resource "launchdarkly_project" "example" {
  key  = "example-project"
  name = "Example project"

  environments = {
    "production" = {
      name  = "Production"
      color = "EEEEEE"
    }
  }
}

# Create a boolean feature flag in that project
resource "launchdarkly_feature_flag" "example" {
  project_key    = launchdarkly_project.example.key
  key            = "example-flag"
  name           = "Example flag"
  variation_type = "boolean"

  variations = [
    { value = "true" },
    { value = "false" },
  ]
}
