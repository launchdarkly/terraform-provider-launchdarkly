data "launchdarkly_big_segment_store_integration" "redis_store" {
  project_key     = "example-project"
  environment_key = "production"
  integration_key = "redis"
  integration_id  = "57c0a8e4f7b9c20b3c5d1234"
}
