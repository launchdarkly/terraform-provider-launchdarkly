resource "launchdarkly_integration_delivery_configuration" "redis_feature_store" {
  project_key     = launchdarkly_project.example.key
  env_key         = "production"
  integration_key = "redis"

  name = "Production Redis feature store"
  on   = true

  config = jsonencode({
    host   = "redis.internal.example.com"
    port   = 6379
    prefix = "launchdarkly"
  })

  tags = ["terraform-managed"]
}
