resource "launchdarkly_context_kind" "organization" {
  project_key = launchdarkly_project.example.key
  key         = "organization"
  name        = "Organization"
}

resource "launchdarkly_experimentation_settings" "example" {
  project_key = launchdarkly_project.example.key
  randomization_units = {
    user = {
      default = true
    }
    organization = {
      default = false
    }
  }

  depends_on = [launchdarkly_context_kind.organization]
}
