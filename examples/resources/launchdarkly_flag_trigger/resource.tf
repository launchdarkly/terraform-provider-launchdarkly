resource "launchdarkly_flag_trigger" "example" {
  project_key     = launchdarkly_project.example.key
  env_key         = "test"
  flag_key        = launchdarkly_feature_flag.trigger_flag.key
  integration_key = "generic-trigger"
  instructions {
    kind = "turnFlagOn"
  }
  enabled = false
}
