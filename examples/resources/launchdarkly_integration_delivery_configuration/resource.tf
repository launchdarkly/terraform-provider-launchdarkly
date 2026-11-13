resource "launchdarkly_integration_delivery_configuration" "fastly_feature_store" {
  project_key     = launchdarkly_project.example.key
  env_key         = "production"
  integration_key = "fastly"

  name = "Production Fastly feature store"
  on   = true

  # The accepted config fields are defined by the integration's manifest.
  # Secret fields such as apiToken are returned obfuscated by the API, so the
  # value you supply here is treated as the source of truth.
  config = jsonencode({
    storeId  = "e6379ca0-17c2-4f58-bb8d-06ea232b01b5"
    apiToken = "my-fastly-api-token"
  })

  tags = ["terraform-managed"]
}
