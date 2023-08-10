# Currently the following five types of destinations are available: kinesis, google-pubsub, mparticle, azure-event-hubs, and segment. Please note that config fields will vary depending on which destination you are trying to configure / access.

resource "launchdarkly_destination" "kinesis_example" {
  project_key = "example-project"
  env_key     = "example-env"
  name        = "example-kinesis-dest"
  kind        = "kinesis"
  config = {
    region      = "us-east-1"
    role_arn    = "arn:aws:iam::123456789012:role/marketingadmin"
    stream_name = "cat-stream"
  }
  on   = true
  tags = ["terraform"]
}

resource "launchdarkly_destination" "pubsub_example" {
  project_key = "example-project"
  env_key     = "example-env"
  name        = "example-pubsub-dest"
  kind        = "google-pubsub"
  config = {
    project = "example-pub-sub-project"
    topic   = "example-topic"
  }
  on   = true
  tags = ["terraform"]
}

resource "launchdarkly_destination" "mparticle_example" {
  project_key = "example-project"
  env_key     = "example-env"
  name        = "example-mparticle-dest"
  kind        = "mparticle"
  config = {
    api_key = "apiKeyfromMParticle"
    secret  = "mParticleSecret"
    user_identities = jsonencode([
      { "ldContextKind" : "user", "mparticleUserIdentity" : "customer_id" },
      { "ldContextKind" : "device", "mparticleUserIdentity" : "google" }]
    )
    environment = "production"
  }
  on   = true
  tags = ["terraform"]
}

resource "launchdarkly_destination" "azure_example" {
  project_key = "example-project"
  env_key     = "example-env"
  name        = "example-azure-event-hubs-dest"
  kind        = "azure-event-hubs"
  config = {
    namespace   = "example-azure-namespace"
    name        = "example-azure-name"
    policy_name = "example-policy-name"
    policy_key  = "azure-event-hubs-policy-key"
  }
  on   = true
  tags = ["terraform"]
}

resource "launchdarkly_destination" "segment_example" {
  project_key = "example-project"
  env_key     = "example-env"
  name        = "example-segment-dest"
  kind        = "segment"
  config = {
    write_key                 = "segment-write-key"
    user_id_context_kind      = "user"
    anonymous_id_context_kind = "anonymousUser"
  }
  on   = true
  tags = ["terraform"]
}
