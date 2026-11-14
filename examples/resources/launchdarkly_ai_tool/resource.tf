resource "launchdarkly_ai_tool" "example" {
  project_key = launchdarkly_project.example.key
  key         = "web-search"
  description = "Search the web for information"
  schema_json = jsonencode({
    type = "object"
    properties = {
      query = {
        type        = "string"
        description = "The search query"
      }
    }
    required = ["query"]
  })
}
