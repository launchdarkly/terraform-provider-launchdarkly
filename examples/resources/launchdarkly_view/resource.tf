resource "launchdarkly_view" "example" {
  project_key = "example-project"
  key         = "example-view"
  name        = "Example View"
  description = "An example view for demonstration purposes"

  tags = [
    "terraform",
    "example"
  ]

  generate_sdk_keys = true
  maintainer_id     = "507f1f77bcf86cd799439011"
}
