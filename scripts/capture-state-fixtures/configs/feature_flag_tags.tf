# Synthetic capture config for launchdarkly_feature_flag with a tag
# set carrying multiple entries.

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
  key  = "fixture-ff-tags-pj"
  name = "Phase 4 ff tags project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_feature_flag" "tags_flag" {
  project_key    = launchdarkly_project.test.key
  key            = "fixture-tags-flag"
  name           = "Phase 4 tags flag fixture"
  variation_type = "boolean"
  tags           = ["fixture", "phase-4", "tag-set"]
  description    = "Flag with multi-tag set"

  variations {
    value = "true"
  }
  variations {
    value = "false"
  }
}
