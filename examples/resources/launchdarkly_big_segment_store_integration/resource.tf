resource "launchdarkly_big_segment_store_integration" "redis_store" {
  project_key     = launchdarkly_project.example.key
  environment_key = "production"
  integration_key = "redis"
  name            = "Production Redis persistent store"
  on              = true

  config = jsonencode({
    host       = "redis.internal.example.com"
    port       = "6379"
    tlsEnabled = true
    username   = "default"
    password   = "example-password"
  })

  tags = ["terraform-managed"]
}

resource "launchdarkly_big_segment_store_integration" "dynamodb_store" {
  project_key     = launchdarkly_project.example.key
  environment_key = "production"
  integration_key = "dynamodb"
  name            = "Production DynamoDB persistent store"
  on              = true

  config = jsonencode({
    tableName  = "launchdarkly-big-segments"
    region     = "us-east-1"
    roleArn    = "arn:aws:iam::123456789012:role/launchdarkly-big-segments"
    externalId = "00000000-0000-0000-0000-000000000000"
  })

  tags = ["terraform-managed"]
}
