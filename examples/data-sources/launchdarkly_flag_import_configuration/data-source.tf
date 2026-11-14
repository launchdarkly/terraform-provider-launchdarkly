data "launchdarkly_flag_import_configuration" "split_import" {
  project_key     = "example-project"
  integration_key = "split"
  integration_id  = "57c12345abcd1234ef567890"
}
