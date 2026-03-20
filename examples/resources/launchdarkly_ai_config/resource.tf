resource "launchdarkly_ai_config" "example" {
  project_key = launchdarkly_project.example.key
  key         = "customer-assistant"
  name        = "Customer Assistant"
  description = "AI assistant for customer support"
  mode        = "completion"
  tags        = ["support"]
}
