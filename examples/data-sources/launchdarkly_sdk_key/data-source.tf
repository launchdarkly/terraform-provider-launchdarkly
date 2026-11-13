data "launchdarkly_sdk_key" "mobile_analytics" {
  project_key     = "example-project"
  environment_key = "production"
  key             = "mobile-analytics-key"
}
