# Synthetic capture config for launchdarkly_segment with non-user
# context targets (included_contexts + excluded_contexts).

terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "2.29.0"
    }
  }
}

provider "launchdarkly" {}

resource "launchdarkly_project" "test" {
  key  = "fixture-segment-ctx-pj"
  name = "Phase 4 segment contexts project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_segment" "contexts" {
  project_key = launchdarkly_project.test.key
  env_key     = "fixture-env-test"
  key         = "fixture-segment-contexts"
  name        = "Phase 4 segment contexts fixture"

  included_contexts {
    context_kind = "organization"
    values       = ["acme", "globex"]
  }
  excluded_contexts {
    context_kind = "organization"
    values       = ["initech"]
  }
}
