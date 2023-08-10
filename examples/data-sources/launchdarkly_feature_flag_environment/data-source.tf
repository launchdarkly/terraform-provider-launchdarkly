data "launchdarkly_feature_flag_environment" "example" {
  flag_id = "example-project/example-flag"
  env_key = "example-env"
}
