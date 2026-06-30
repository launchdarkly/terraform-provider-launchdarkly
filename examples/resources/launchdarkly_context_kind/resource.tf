resource "launchdarkly_project" "example" {
  key  = "example-project"
  name = "Example Project"
  environments = {
    "production" = {
      key   = "production"
      name  = "Production"
      color = "000000"
    }
  }
}

resource "launchdarkly_context_kind" "organization" {
  project_key = launchdarkly_project.example.key
  key         = "organization"
  name        = "Organization"
  description = "An organization that owns one or more accounts"
}
