resource "launchdarkly_release_policy" "guarded_example" {
  project_key    = "example-project"
  key            = "production-guarded"
  name           = "Production Guarded Release"
  release_method = "guarded-release"

  # Optional: Add scope configuration 
  scope {
    environment_keys = ["production", "staging"]
  }

  # Required for guarded-release method
  guarded_release_config {
    rollback_on_regression = true
    min_sample_size        = 100
  }
}

resource "launchdarkly_release_policy" "progressive_example" {
  project_key    = "example-project"
  key            = "staging-progressive"
  name           = "Staging Progressive Release"
  release_method = "progressive-release"
}

# To import an existing release policy, use:
# terraform import launchdarkly_release_policy.guarded_example example-project/production-guarded
