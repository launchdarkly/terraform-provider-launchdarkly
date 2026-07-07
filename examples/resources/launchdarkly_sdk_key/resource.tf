resource "launchdarkly_sdk_key" "mobile_analytics" {
  project_key     = launchdarkly_project.example.key
  environment_key = "production"
  key             = "mobile-analytics-key"
  name            = "Mobile analytics SDK key"
  kind            = "mobile"
  description     = "SDK key used by the mobile analytics service"
}
