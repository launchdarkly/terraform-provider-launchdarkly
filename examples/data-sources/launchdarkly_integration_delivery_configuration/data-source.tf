data "launchdarkly_integration_delivery_configuration" "redis_feature_store" {
  project_key     = "example-project"
  env_key         = "production"
  integration_key = "redis"
  config_id       = "57c1e8b1b8e8c50c3f000001"
}
