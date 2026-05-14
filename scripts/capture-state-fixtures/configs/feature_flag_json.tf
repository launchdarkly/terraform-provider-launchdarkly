# Synthetic capture config for launchdarkly_feature_flag json
# variation type. Compact JSON values to satisfy v2.29 plan-apply
# consistency checks (state-compat capture playbook gotcha).

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
  key  = "fixture-ff-json-pj"
  name = "Phase 4 ff json project"
  environments {
    key   = "fixture-env-test"
    name  = "Fixture test env"
    color = "112233"
  }
}

resource "launchdarkly_feature_flag" "json_flag" {
  project_key    = launchdarkly_project.test.key
  key            = "fixture-json-flag"
  name           = "Phase 4 json flag fixture"
  variation_type = "json"

  variations {
    value = "{\"version\":1}"
  }
  variations {
    value = "{\"version\":2}"
  }
}
