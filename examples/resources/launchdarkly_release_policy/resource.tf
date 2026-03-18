resource "launchdarkly_release_policy" "guarded_example" {
  project_key    = "example-project"
  key            = "production-guarded"
  name           = "Production Guarded Release"
  release_method = "guarded-release"

  scope {
    environment_keys = ["production", "staging"]
  }

  guarded_release_config {
    rollback_on_regression = true
    min_sample_size        = 100

    stages {
      allocation      = 25000
      duration_millis = 60000
    }
    stages {
      allocation      = 50000
      duration_millis = 0
    }
  }
}

resource "launchdarkly_release_policy" "progressive_example" {
  project_key    = "example-project"
  key            = "staging-progressive"
  name           = "Staging Progressive Release"
  release_method = "progressive-release"

  progressive_release_config {
    stages {
      allocation      = 25000
      duration_millis = 60000
    }
    stages {
      allocation      = 50000
      duration_millis = 120000
    }
    stages {
      allocation      = 100000
      duration_millis = 0
    }
  }
}

# To import an existing release policy, use:
# terraform import launchdarkly_release_policy.guarded_example example-project/production-guarded
