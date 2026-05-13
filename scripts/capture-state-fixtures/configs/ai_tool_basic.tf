terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "2.29.0"
    }
  }
}

resource "launchdarkly_project" "fixture" {
  key  = "fixture-project-1"
  name = "fixture-project-1"
  environments {
    key   = "fixture-env-test"
    name  = "fixture-env-test"
    color = "AABBCC"
  }
}

resource "launchdarkly_ai_tool" "fixture" {
  project_key = launchdarkly_project.fixture.key
  key         = "fixture-ai-tool-basic"
  description = "fixture ai tool"
  schema_json = jsonencode({
    type = "object"
    properties = {
      query = { type = "string" }
    }
    required = ["query"]
  })
  # SDKv2 v2.29 produces inconsistent null vs empty-string apply result
  # for unset Optional+Computed JSON strings; set explicitly to "{}".
  custom_parameters = "{}"
}
