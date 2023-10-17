resource "launchdarkly_segment" "example" {
  key         = "example-segment-key"
  project_key = launchdarkly_project.example.key
  env_key     = launchdarkly_environment.example.key
  name        = "example segment"
  description = "This segment is managed by Terraform"
  tags        = ["segment-tag-1", "segment-tag-2"]
  included    = ["user1", "user2"]
  excluded    = ["user3", "user4"]
  included_contexts {
    values       = ["account1", "account2"]
    context_kind = "account"
  }

  rules {
    clauses {
      attribute    = "country"
      op           = "startsWith"
      values       = ["en", "de", "un"]
      negate       = false
      context_kind = "location-data"
    }
  }
}

resource "launchdarkly_segment" "big-example" {
  key                    = "example-big-segment-key"
  project_key            = launchdarkly_project.example.key
  env_key                = launchdarkly_environment.example.key
  name                   = "example big segment"
  description            = "This big segment is managed by Terraform"
  tags                   = ["segment-tag-1", "segment-tag-2"]
  unbounded              = true
  unbounded_context_kind = "user"
}
