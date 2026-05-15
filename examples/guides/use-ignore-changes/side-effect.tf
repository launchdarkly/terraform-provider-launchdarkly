resource "launchdarkly_feature_flag" "example" {
  project_key = launchdarkly_project.example.key
  key         = "example-flag"
  name        = "Example flag"
  description = "This demonstrates using ignore_changes"

  variation_type = "boolean"
  variations {
    value = "true"
    name  = "True"
  }
  variations {
    value = "false"
    name  = "False"
  }

  defaults {
    on_variation  = 1
    off_variation = 0
  }

  lifecycle {
    ignore_changes = [all]
  }
}
