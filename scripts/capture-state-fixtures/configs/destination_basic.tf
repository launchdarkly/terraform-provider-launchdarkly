# Synthetic capture config for launchdarkly_destination (Phase 3.1).
# Uses the segment kind because it requires only a write_key, which the
# scan.sh fixture-safety regex tolerates as the deterministic
# fixture-token-PLACEHOLDER value. Other kinds (mparticle / kinesis)
# need real-looking creds that would trip the secret scanner; capture
# them in a separate fixture once the sanitiser knows about each
# config-key.

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
  key  = "fixture-project-1"
  name = "Phase 3 destination fixture"

  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_destination" "basic" {
  project_key = launchdarkly_project.test.key
  env_key     = "fixture-env-test"
  name        = "fixture-segment-destination"
  kind        = "segment"
  on          = false

  config = {
    write_key = "fixture-token-PLACEHOLDER"
  }

  tags = ["fixture"]
}
