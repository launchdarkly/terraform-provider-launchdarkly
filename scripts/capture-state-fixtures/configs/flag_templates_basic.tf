# Synthetic capture config for launchdarkly_flag_templates (Phase 3.10).
# The resource is a singleton per project: Create + Update both PUT
# /flag-defaults; Delete is a no-op (templates always exist).

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
  name = "Phase 3 flag_templates fixture"

  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_flag_templates" "basic" {
  project_key = launchdarkly_project.test.key
  temporary   = false
  tags        = ["fixture"]

  boolean_defaults {
    true_display_name  = "On"
    false_display_name = "Off"
    true_description   = "Fixture true variation"
    false_description  = "Fixture false variation"
    on_variation       = 0
    off_variation      = 1
  }
}
