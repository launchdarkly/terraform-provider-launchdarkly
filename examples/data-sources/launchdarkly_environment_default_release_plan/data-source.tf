data "launchdarkly_environment_default_release_plan" "production" {
  project_key     = "example-project"
  environment_key = "production"
}
