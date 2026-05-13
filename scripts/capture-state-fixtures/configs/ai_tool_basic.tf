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
}

resource "launchdarkly_ai_tool" "fixture" {
  project_key = launchdarkly_project.fixture.key
  key         = "fixture-ai-tool-basic"
  name        = "fixture-ai-tool-basic"
  description = "fixture ai tool"
  schema_json = jsonencode({
    type = "object"
    properties = {
      query = { type = "string" }
    }
    required = ["query"]
  })
}
